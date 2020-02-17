package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var getCredCmd = &cobra.Command{
	Use:   "get-cred <OIDC provider name>",
	Short: "Get AWS credentials and out to stdout",
	Long:  `Get AWS credentials and out to stdout through your OIDC provider authentication.`,
	Run:   getCred,
}

type AWSCredentials struct {
	AWSAccessKey     string
	AWSSecretKey     string
	AWSSessionToken  string
	AWSSecurityToken string
	PrincipalARN     string
	Expires          time.Time
}

func init() {
	rootCmd.AddCommand(getCredCmd)
}

func getCred(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		Writeln("The OIDC provider name is required")
		Exit(nil)
	}
	providerName := args[0]

	client, err := CheckInstalled(providerName)
	if err != nil {
		Writeln("Failed to login OIDC provider")
		Exit(err)
	}

	tokenResponse, err := doLogin(client)
	if err != nil {
		Writeln("Failed to login the OIDC provider")
		Exit(err)
	}

	idToken := tokenResponse.Extra("id_token").(string)
	Writeln("Login successful!")
	Traceln("ID token: %s", idToken)

	maxSessionDurationSecondsString := client.config.GetString(MaxSessionDurationSeconds)
	maxSessionDurationSeconds, err := strconv.ParseInt(maxSessionDurationSecondsString, 10, 64)
	if err != nil {
		maxSessionDurationSeconds = 3600
	}

	awsCreds, err := GetCredentialsWithOIDC(client, idToken, maxSessionDurationSeconds)
	if err != nil {
		fmt.Printf("Unable to get AWS Credentials: %v\n", err)
	}

	Writeln("")

	type AWSCredentialsJSON struct {
		Version         int
		AccessKeyID     string
		SecretAccessKey string
		SessionToken    string
	}

	awsCredsJSON := AWSCredentialsJSON{
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

func doLogin(client *OIDCClient) (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, errors.Wrap(err, "Cannot start local http server to handle login redirect")
	}
	port := listener.Addr().(*net.TCPAddr).Port

	redirect := fmt.Sprintf("http://127.0.0.1:%d", port)

	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     client.config.GetString(ClientID),
		ClientSecret: client.config.GetString(ClientSecret),
		Endpoint:     endpoints.Google,
		RedirectURL:  redirect,
		Scopes:       []string{"openid", "email"},
	}

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
