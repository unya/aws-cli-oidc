package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

type SingleCache struct {
	mu         sync.Mutex        `json:-`
	Id         string            `json:"id"`
	OidcTokens map[string]string `json:"oidc"`
	AwsTokens  map[string]string `json:"aws"`
}

func init() {
	SingletonCache.AwsTokens = make(map[string]string)
	SingletonCache.OidcTokens = make(map[string]string)
	SingletonCache.Load()
}

var ErrNotFound = errors.New("cache entry not found")

var keyringServiceName = "aws-cli-oidc"
var keyringUsername = os.Getenv("USER")
var keyringServiceNameAWS = keyringServiceName + "-aws"
var keyringServiceNameOIDC = keyringServiceName + "-oidc"

var SingletonCache SingleCache

func (cache *SingleCache) Load() error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	jsonString, err := keyring.Get(keyringServiceName, keyringUsername)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil
		}
		return err
	}
	err = json.Unmarshal([]byte(jsonString), &cache)
	return err
}

func (cache *SingleCache) Save() error {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	jsonString, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	err = keyring.Set(keyringServiceName, keyringUsername, string(jsonString))
	return err
}

func getAWSTokenCache(role string) (string, error) {
	token, err := SingletonCache.AwsTokens[role]
	if err == false {
		return "", ErrNotFound
	}
	return token, nil
}

func saveAWSTokenCache(awsCredsJSON string, role string) error {
	SingletonCache.AwsTokens[role] = awsCredsJSON
	return SingletonCache.Save()
}

func getOIDCTokenCache(role string) (string, error) {
	token, err := SingletonCache.OidcTokens[role]
	if err == false {
		return "", ErrNotFound
	}
	return token, nil
}

func saveOIDCTokenCache(awsCredsJSON string, role string) error {
	SingletonCache.OidcTokens[role] = awsCredsJSON
	return SingletonCache.Save()
}

func CacheShow(showToken bool) (string, error) {
	var response strings.Builder

	err := SingletonCache.Load()
	if err != nil {
		return "", err
	}

	SingletonCache.mu.Lock()
	defer SingletonCache.mu.Unlock()
	response.WriteString(fmt.Sprintf("OIDC Tokens for %s:\n", keyringUsername))
	for role, token := range SingletonCache.OidcTokens {
		var decodedToken oidcToken
		err := json.Unmarshal([]byte(token), &decodedToken)
		if err != nil {
			continue
		}
		if showToken {
			response.WriteString(fmt.Sprintf("[%s]: %v\n", role, token))
		} else {
			response.WriteString(fmt.Sprintf("\t[%s]: %v\n", role, decodedToken.Expiry))
		}
	}
	response.WriteString(fmt.Sprintf("AWS Tokens for %s:\n", keyringUsername))
	for role, token := range SingletonCache.AwsTokens {
		var decodedToken AWSCredentials
		err := json.Unmarshal([]byte(token), &decodedToken)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if showToken {
			response.WriteString(fmt.Sprintf("[%s]: %v\n", role, token))
		} else {
			response.WriteString(fmt.Sprintf("\t[%s]: %v\n", role, decodedToken.Expires))
		}
	}

	return response.String(), nil
}

func CacheClear() error {
	keyring.Delete(keyringServiceName, keyringUsername)
	return nil
}
