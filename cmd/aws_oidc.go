package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentity"
	"github.com/pkg/errors"
)

const awsCredentialsCache = "aws_token.json"
const expiryDelta = 10 * time.Second

type AWSCredentials struct {
	AWSAccessKey     string
	AWSSecretKey     string
	AWSSessionToken  string
	AWSSecurityToken string
	PrincipalARN     string
	Expires          time.Time
}

func (cred AWSCredentials) Valid() bool {
	if cred.Expires.IsZero() {
		return false
	}
	return !cred.Expires.Add(-expiryDelta).Before(time.Now())
}

func GetCredentialsWithOIDC(client *OIDCClient, idToken string, durationSeconds int64) (*AWSCredentials, error) {
	jsonBytes, err := ioutil.ReadFile(awsCredentialsCache)
	var awsCreds *AWSCredentials = nil
	if err != nil {
		if !os.IsNotExist(err) { // if file does not exist we are fine
			return nil, err
		}
	} else {
		if err := json.Unmarshal(jsonBytes, &awsCreds); err != nil {
			return nil, err
		}
	}

	if awsCreds != nil && awsCreds.Valid() {
		return awsCreds, nil
	}

	creds, err := getCredentialsUsingIDToken(client, idToken, durationSeconds)
	if err != nil {
		return nil, err
	}

	awsCredsJSON, err := json.Marshal(creds)
	if err != nil {
		return nil, err
	}

	file, err := ioutil.TempFile("", "*")
	if err != nil {
		return nil, err
	}

	_, err = file.Write(awsCredsJSON)
	if err != nil {
		return nil, err
	}

	err = os.Rename(file.Name(), awsCredentialsCache)
	if err != nil {
		return nil, err
	}

	return creds, err
}

func getCredentialsUsingIDToken(client *OIDCClient, idToken string, durationSeconds int64) (*AWSCredentials, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}

	identityPoolID := client.config.GetString(IdentityPoolID)
	split := strings.SplitN(identityPoolID, ":", 2)
	if len(split) != 2 {
		return nil, errors.New("Identity pool ID does not contain the region")
	}
	region := split[0]

	login := map[string]*string{}
	login[client.config.GetString(OIDCServer)] = &idToken

	cognitoIdentity := cognitoidentity.New(sess, aws.NewConfig().WithRegion(region))

	idResp, err := cognitoIdentity.GetId(&cognitoidentity.GetIdInput{
		IdentityPoolId: &identityPoolID,
		Logins:         login,
	})
	if err != nil {
		fmt.Printf("Error retrieving GetId: %s", err)
		return nil, err
	}

	credsResp, err := cognitoIdentity.GetCredentialsForIdentity(&cognitoidentity.GetCredentialsForIdentityInput{
		IdentityId: idResp.IdentityId,
		Logins:     login,
	})
	if err != nil {
		fmt.Printf("Error retrieving GetCredentialsForIdentity: %s", err)
		return nil, err
	}

	return &AWSCredentials{
		AWSAccessKey:    aws.StringValue(credsResp.Credentials.AccessKeyId),
		AWSSecretKey:    aws.StringValue(credsResp.Credentials.SecretKey),
		AWSSessionToken: aws.StringValue(credsResp.Credentials.SessionToken),
		Expires:         credsResp.Credentials.Expiration.Local(),
	}, nil
}
