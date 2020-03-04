package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/docopt/docopt-go"
	"github.com/mattn/go-shellwords"
	"github.com/mbrtargeting/aws-cli-oidc/internal"
	"github.com/pkg/browser"
	"gopkg.in/ini.v1"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	usage := `aws-sign-in.

Usage:
  aws-sign-in [<profile>]
  aws-sign-in -h | --help

Options:
  -h --help  Show this screen.`

	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	var conf struct {
		Profile string `docopt:"<profile>"`
	}
	if err := arguments.Bind(&conf); err != nil {
		log.Fatalf("%v\n", err)
	}

	profileName := conf.Profile
	if profileName == "" {
		profileName = os.Getenv("AWS_PROFILE")
		if profileName == "" {
			profileName = "default"
		}
	}

	configPath, credentialsPath := getAWSConfigPaths()

	creds, err := runCredentialProcess(configPath, credentialsPath, profileName)
	if err != nil {
		log.Fatalf("Error running the credential process: %v\n", err)
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

func getAWSConfigPaths() (string, string) {
	var credentialsPath string
	var configPath string

	awsConfigFile := os.Getenv("AWS_CONFIG_FILE")
	if awsConfigFile != "" {
		configPath = awsConfigFile
	}

	awsCredentialsFile := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if awsCredentialsFile != "" {
		credentialsPath = awsCredentialsFile
	}

	home, err := os.UserHomeDir()
	if err == nil {
		// only overwrite the paths if the respective env var was not set
		if configPath == "" {
			configPath = home + "/.aws/config"
		}
		if credentialsPath == "" {
			credentialsPath = home + "/.aws/credentials"
		}
	}

	return configPath, credentialsPath
}

func tryRunCredentialProcess(credentialProcess string) (*internal.AWSCredentialsJSON, error) {
	words, err := shellwords.Parse(credentialProcess)
	if err != nil || len(words) == 0 {
		return nil, fmt.Errorf("invalid credential_process entry")
	}
	cmd := exec.Command(words[0], words[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run the credential process: %v", err)
	}

	var creds internal.AWSCredentialsJSON
	err = json.Unmarshal(out.Bytes(), &creds)
	if err != nil {
		return nil, fmt.Errorf("error parsing credential process output: %v", err)
	}

	return &creds, nil
}

func findCredentialProcess(path string, profile string) (string, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	section, err := cfg.GetSection(profile)
	if err != nil {
		return "", fmt.Errorf("failed to read section: %v", err)
	}

	key, err := section.GetKey("credential_process")
	if err != nil {
		return "", fmt.Errorf("failed to find credential_process entry: %v", err)
	}

	return key.String(), nil
}

func runCredentialProcess(configPath string, credentialsPath string, profile string) (*internal.AWSCredentialsJSON, error) {
	credentialProcess, err := findCredentialProcess(credentialsPath, profile)
	if err == nil {
		return tryRunCredentialProcess(credentialProcess)
	}

	// yes, the ~/.aws/config has a different naming scheme for profile names (must be prefixed with "profile")
	configProfileName := profile
	if profile != "default" {
		configProfileName = fmt.Sprintf("profile %s", profile)
	}
	credentialProcess, err = findCredentialProcess(configPath, configProfileName)
	if err == nil {
		return tryRunCredentialProcess(credentialProcess)
	}

	return nil, fmt.Errorf("not able to find a valid credential_process")
}

func getSignInToken(creds *internal.AWSCredentialsJSON) (string, error) {
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
