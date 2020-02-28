package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

type oidcToken struct {
	*oauth2.Token
	IDToken string `json:"id_token,omitempty"`
}

func oidcTokenFromOAuth2Token(token *oauth2.Token) *oidcToken {
	oidcToken := &oidcToken{
		Token:   token,
		IDToken: token.Extra("id_token").(string),
	}
	return oidcToken
}

func (t oidcToken) OAuth2Token() *oauth2.Token {
	return t.WithExtra(map[string]interface{}{
		"id_token": t.IDToken,
	})
}

func getCred(providerName string, roleARN string) {
	client, err := CheckInstalled(providerName)
	if err != nil {
		log.Fatalf("Failed to login OIDC provider: %v\n", err)
	}

	tokenResponse, err := getOIDCToken(client)
	if err != nil {
		log.Fatalln("Failed to login the OIDC provider")
	}

	log.Println("Login successful!")
	log.Printf("ID token: %s\n", tokenResponse.IDToken)

	maxSessionDurationSecondsString := client.config.MaxSessionDurationSeconds
	maxSessionDurationSeconds, err := strconv.ParseInt(maxSessionDurationSecondsString, 10, 64)
	if err != nil {
		maxSessionDurationSeconds = 3600
	}

	awsCreds, err := GetCredentialsWithOIDC(client, tokenResponse.IDToken, roleARN, maxSessionDurationSeconds)
	if err != nil {
		log.Fatalf("Unable to get AWS Credentials: %v\n", err)
	}

	type awsCredentialsJSON struct {
		Version         int
		AccessKeyID     string `json:"AccessKeyId"`
		SecretAccessKey string
		SessionToken    string
	}

	awsCredsJSON := awsCredentialsJSON{
		Version:         1,
		AccessKeyID:     awsCreds.AWSAccessKey,
		SecretAccessKey: awsCreds.AWSSecretKey,
		SessionToken:    awsCreds.AWSSessionToken,
	}

	jsonBytes, err := json.Marshal(awsCredsJSON)
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Stdout.Write(jsonBytes)
}

func getOIDCToken(client *OIDCClient) (*oidcToken, error) {
	oidcTokenCache := ConfigPath() + "/" + client.name + "_oidc.json"
	conf := &oauth2.Config{
		ClientID:     client.config.ClientID,
		ClientSecret: client.config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  client.config.AuthURL,
			TokenURL: client.config.TokenURL,
		},
		RedirectURL: "",
		Scopes:      []string{"openid", "email"},
	}

	var token *oauth2.Token
	writeBack := false

	var oidcToken *oidcToken = nil
	jsonRaw, err := ioutil.ReadFile(oidcTokenCache)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(jsonRaw, &oidcToken); err != nil {
			return nil, err
		}
	}

	if oidcToken != nil { // cache hit
		token = oidcToken.OAuth2Token()

		if !token.Valid() {
			writeBack = true

			tokenSource := conf.TokenSource(context.Background(), token)
			token, err = tokenSource.Token()
			// If we get an error here, we assume that the refresh token expired. Since token remains nil, the next
			// step will trigger a login flow.
			_ = err
		}
	}

	if token == nil { // cache miss or expired refresh token
		writeBack = true

		token, err = doLogin(conf)
		if err != nil {
			return nil, err
		}
	}

	oidcToken = oidcTokenFromOAuth2Token(token)

	if writeBack {
		tokenJSON, _ := json.Marshal(oidcToken)

		file, err := ioutil.TempFile("", "*")
		if err != nil {
			return nil, err
		}

		_, err = file.Write(tokenJSON)
		if err != nil {
			return nil, err
		}

		err = os.Rename(file.Name(), oidcTokenCache)
		if err != nil {
			return nil, err
		}
	}

	return oidcToken, nil
}

func doLogin(conf *oauth2.Config) (*oauth2.Token, error) {
	address := "localhost:52327"
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot start local http server to handle login redirect")
	}

	conf.RedirectURL = "http://" + address

	ctx := context.Background()

	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	println(url)

	code := launch(url, listener)

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}

	return tok, err
}

func launch(url string, listener net.Listener) string {
	c := make(chan string)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		url := req.URL
		q := url.Query()
		code := q.Get("code")

		res.Header().Set("Content-Type", "text/html")

		// Response result page
		message := "Login "
		if code != "" {
			message += "successful"
		} else {
			message += "failed"
		}
		res.Header().Set("Cache-Control", "no-store")
		res.Header().Set("Pragma", "no-cache")
		res.WriteHeader(200)
		_, _ = res.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
<body>
%s
</body>
</html>
`, message)))

		if f, ok := res.(http.Flusher); ok {
			f.Flush()
		}

		time.Sleep(100 * time.Millisecond)

		c <- code
	})

	srv := &http.Server{}
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		_ = srv.Shutdown(ctx)
	}()

	go func() {
		_ = srv.Serve(listener)
	}()

	var code string
	if err := browser.OpenURL(url); err == nil {
		code = <-c
	}

	return code
}
