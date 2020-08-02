package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/docopt/docopt-go"
	"github.com/pkg/browser"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	usage := `aws-sign-in.

Usage:
  aws-sign-in
  aws-sign-in -h | --help

Options:
  -h --help  Show this screen.`

	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	var conf struct {
	}
	if err := arguments.Bind(&conf); err != nil {
		log.Fatalf("%v\n", err)
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		fmt.Printf("unable to load SDK config: %v\n", err)
	}
	cfg.Region = "eu-central-1"

	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		log.Fatalf("Error retrieving credentials: %v\n", err)
	}

	signinToken, err := getSignInToken(creds)
	if err != nil {
		log.Fatalf("Couldn't obtain sign-in token: %v\n", err)
	}

	loginURL := constructLoginURL(signinToken)

	// we just ignore the error and display the following message regardless
	// this way, we also protect us from the case where the browser fails to open, but then OpenURL call returns no error
	_ = browser.OpenURL(loginURL)
	fmt.Printf("If the browser didn't open, please visit the following url to sign in to the AWS console: %v\n", loginURL)
}

func getSignInToken(creds aws.Credentials) (string, error) {
	reqJSON := fmt.Sprintf(`{"sessionId":"%s","sessionKey":"%s","sessionToken":"%s"}`,
		creds.AccessKeyID,
		creds.SecretAccessKey,
		creds.SessionToken)

	// since this is a static string, we know for sure there will be no error
	endpoint, _ := url.Parse("https://signin.aws.amazon.com/federation")

	values := url.Values{}
	values.Add("Action", "getSigninToken")
	values.Add("Session", reqJSON)

	endpoint.RawQuery = values.Encode()

	response, err := http.Get(endpoint.String())
	if err != nil {
		return "", fmt.Errorf("error during signin: %v", err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	var respJSON map[string]string
	err = json.Unmarshal(body, &respJSON)
	if err != nil {
		return "", fmt.Errorf("error parsing response as JSON: %v", err)
	}

	signinToken, exists := respJSON["SigninToken"]
	if !exists {
		return "", fmt.Errorf("response does not contain sign-in token")
	}

	return signinToken, nil
}

func constructLoginURL(signinToken string) string {
	// since this is a static string, we know for sure there will be no error
	consoleEndpoint, _ := url.Parse("https://signin.aws.amazon.com/federation")

	consoleValues := url.Values{}
	consoleValues.Add("Action", "login")
	consoleValues.Add("Destination", "https://console.aws.amazon.com")
	consoleValues.Add("SigninToken", signinToken)

	consoleEndpoint.RawQuery = consoleValues.Encode()

	return consoleEndpoint.String()
}
