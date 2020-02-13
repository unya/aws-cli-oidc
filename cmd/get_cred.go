package cmd

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

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

type LoginParams struct {
	ResponseType string `url:"response_type,omitempty"`
	ClientId     string `url:"client_id,omitempty"`
	RedirectUri  string `url:"redirect_uri,omitempty"`
	Display      string `url:"display,omitempty"`
	Scope        string `url:"scope,omitempty"`
}

type param struct {
	name  string
	label string
	mask  bool
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

	awsFedType := client.config.GetString(AWS_FEDERATION_TYPE)
	maxSessionDurationSecondsString := client.config.GetString(MAX_SESSION_DURATION_SECONDS)
	maxSessionDurationSeconds, err := strconv.ParseInt(maxSessionDurationSecondsString, 10, 64)
	if err != nil {
		maxSessionDurationSeconds = 3600
	}

	var awsCreds *AWSCredentials
	if awsFedType == AWS_FEDERATION_TYPE_OIDC {
		awsCreds, err = GetCredentialsWithOIDC(client, idToken, maxSessionDurationSeconds)
		if err != nil {
			Writeln("Failed to get aws credentials with OIDC")
			Exit(err)
		}
	} else {
		Writeln("Invalid AWS federation type")
		Exit(err)
	}

	Writeln("")

	Export("AWS_ACCESS_KEY_ID", awsCreds.AWSAccessKey)
	Export("AWS_SECRET_ACCESS_KEY", awsCreds.AWSSecretKey)
	Export("AWS_SESSION_TOKEN", awsCreds.AWSSessionToken)
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
		ClientID:     client.config.GetString(CLIENT_ID),
		ClientSecret: client.config.GetString(CLIENT_SECRET),
		Endpoint:     google.Endpoint,
		RedirectURL:  redirect,
		Scopes:       []string{"openid", "email"},
	}

	url := conf.AuthCodeURL("state")

	code := launch(client, url, listener)

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}

	return tok, err
}

func launch(client *OIDCClient, url string, listener net.Listener) string {
	c := make(chan string)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		url := req.URL
		q := url.Query()
		code := q.Get("code")

		res.Header().Set("Content-Type", "text/html")

		// Redirect to user-defined successful/failure page
		successful := client.RedirectToSuccessfulPage()
		if successful != nil && code != "" {
			url := successful.Url()
			res.Header().Set("Location", (&url).String())
			res.WriteHeader(302)
		}
		failure := client.RedirectToFailurePage()
		if failure != nil && code == "" {
			url := failure.Url()
			res.Header().Set("Location", (&url).String())
			res.WriteHeader(302)
		}

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
		res.Write([]byte(fmt.Sprintf(`<!DOCTYPE html>
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
	defer srv.Shutdown(ctx)

	go func() {
		if err := srv.Serve(listener); err != nil {
			// cannot panic, because this probably is an intentional close
		}
	}()

	var code string
	if err := browser.OpenURL(url); err == nil {
		code = <-c
	}

	return code
}
