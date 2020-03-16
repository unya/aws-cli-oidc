package internal

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

var ErrNotFound = errors.New("cache entry not found")

var keyringServiceName = "aws-cli-oidc"
var keyringUsername = os.Getenv("USER")
var keyringServiceNameAWS = keyringServiceName + "-aws"
var keyringServiceNameOIDC = keyringServiceName + "-oidc"

func getTokenCache(serviceName string) (string, error) {
	jsonString, err := keyring.Get(serviceName, keyringUsername)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", ErrNotFound
		}
		return "", err
	}
	return jsonString, nil
}

func saveTokenCache(serviceName string, awsCredsJSON string) error {
	if err := keyring.Set(serviceName, keyringUsername, string(awsCredsJSON)); err != nil {
		return err
	}
	return nil
}

func getAWSTokenCache() (string, error) {
	return getTokenCache(keyringServiceNameAWS)
}

func saveAWSTokenCache(awsCredsJSON string) error {
	return saveTokenCache(keyringServiceNameAWS, awsCredsJSON)
}

func getOIDCTokenCache() (string, error) {
	return getTokenCache(keyringServiceNameOIDC)
}

func saveOIDCTokenCache(awsCredsJSON string) error {
	return saveTokenCache(keyringServiceNameOIDC, awsCredsJSON)
}

func CacheShow() (string, error) {
	var response strings.Builder

	for _, service := range []string{keyringServiceNameAWS, keyringServiceNameOIDC} {
		response.WriteString(fmt.Sprintf("[%s,%s]: ", service, keyringUsername))
		cache, err := getTokenCache(service)
		if err != nil {
			if err != ErrNotFound {
				return "", err
			}
			response.WriteString("<not set>\n")
			continue
		}
		response.WriteString(cache)
		response.WriteByte('\n')
	}

	return response.String(), nil
}

func CacheClear() error {
	for _, service := range []string{keyringServiceNameAWS, keyringServiceNameOIDC} {
		err := keyring.Delete(service, keyringUsername)
		if err != nil && err != keyring.ErrNotFound {
			return err
		}
	}
	return nil
}
