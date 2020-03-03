package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

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

func GetCredentialsWithOIDC(client *OIDCClient, idToken string, roleARN string, durationSeconds int64) (*AWSCredentials, error) {
	awsCredentialsCache := ConfigPath() + "/" + client.name + "_aws.json"

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

	token, err := assumeRoleWithWebIdentity(client, idToken, roleARN, durationSeconds)
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

func assumeRoleWithWebIdentity(client *OIDCClient, idToken string, roleARN string, durationSeconds int64) (*AWSCredentials, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}

	svc := sts.New(sess)

	username := os.Getenv("USER")
	split := strings.SplitN(roleARN, "/", 2)
	rolename := client.name
	if len(split) == 2 {
		rolename = split[1]
	}

	log.Println("Requesting AWS credentials using ID Token")

	resp, err := svc.AssumeRoleWithWebIdentity(&sts.AssumeRoleWithWebIdentityInput{
		RoleArn:          aws.String(roleARN),
		RoleSessionName:  aws.String(username + "@" + rolename),
		WebIdentityToken: aws.String(idToken),
		DurationSeconds:  aws.Int64(durationSeconds),
	})
	if err != nil {
		return nil, fmt.Errorf("error retrieving STS credentials using ID Token: %v", err)
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
