package cmd

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
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

	token, err := loginToStsUsingIDToken(client, idToken, durationSeconds)
	if err != nil {
		return nil, err
	}

	awsCredsJSON, err := json.Marshal(token)
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

	return token, err
}

func loginToStsUsingIDToken(client *OIDCClient, idToken string, durationSeconds int64) (*AWSCredentials, error) {
	role := client.config.GetString(AwsFederationRole)
	roleSessionName := client.config.GetString(AwsFederationRoleSessionName)

	sess, err := session.NewSession()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create session")
	}

	svc := sts.New(sess)

	params := &sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          &role,
		RoleSessionName:  &roleSessionName,
		WebIdentityToken: &idToken,
		DurationSeconds:  aws.Int64(durationSeconds),
	}

	Writeln("Requesting AWS credentials using ID Token")

	resp, err := svc.AssumeRoleWithWebIdentity(params)
	if err != nil {
		return nil, errors.Wrap(err, "Error retrieving STS credentials using ID Token")
	}

	return &AWSCredentials{
		AWSAccessKey:     aws.StringValue(resp.Credentials.AccessKeyId),
		AWSSecretKey:     aws.StringValue(resp.Credentials.SecretAccessKey),
		AWSSessionToken:  aws.StringValue(resp.Credentials.SessionToken),
		AWSSecurityToken: aws.StringValue(resp.Credentials.SessionToken),
		PrincipalARN:     aws.StringValue(resp.AssumedRoleUser.Arn),
		Expires:          resp.Credentials.Expiration.Local(),
	}, nil
}
