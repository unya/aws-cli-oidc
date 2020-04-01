package internal

import (
	"encoding/json"
	"fmt"
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
	awsCredsBag := map[string]*AWSCredentials{}
	jsonString, err := getAWSTokenCache()
	if err != nil {
		if err != ErrNotFound {
			return nil, err
		}
	} else {
		if err := json.Unmarshal([]byte(jsonString), &awsCredsBag); err != nil {
			return nil, err
		}
	}

	awsCreds, awsCredsFound := awsCredsBag[roleARN]
	if awsCredsFound && awsCreds.Valid() {
		return awsCreds, nil
	}

	token, err := assumeRoleWithWebIdentity(client, idToken, roleARN, durationSeconds)
	if err != nil {
		return nil, err
	}

	awsCredsBag[roleARN] = token
	awsCredsBagJSON, err := json.Marshal(awsCredsBag)
	if err != nil {
		return nil, err
	}

	if err := saveAWSTokenCache(string(awsCredsBagJSON)); err != nil {
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
