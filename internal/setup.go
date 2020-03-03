package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	input "github.com/natsukagami/go-input"
)

func RunSetup(providerName string) error {
	_, err := runSetup(providerName)
	return err
}

func runSetup(providerName string) (*providerConfig, error) {
	ui := &input.UI{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}

	var authURL string
	var tokenURL string
	oidcServer, _ := ui.Ask("OIDC provider metadata server name (https://<server>/.well-known/openid-configuration):", &input.Options{
		Required: true,
		Loop:     true,
		ValidateFunc: func(s string) error {
			u, err := url.Parse(s)
			if err != nil {
				return err
			}

			u.Path = path.Join(u.Path, ".well-known", "openid-configuration")
			u.Scheme = "https"
			res, err := http.Get(u.String())
			if err != nil {
				return err
			}

			type oidcMetadata struct {
				AuthURL  string `json:"authorization_endpoint"`
				TokenURL string `json:"token_endpoint"`
			}

			bytes, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return err
			}

			var meta oidcMetadata
			if err := json.Unmarshal(bytes, &meta); err != nil {
				return err
			}

			authURL = meta.AuthURL
			tokenURL = meta.TokenURL
			return nil
		},
	})
	clientID, _ := ui.Ask("Client ID which is registered in the OIDC provider:", &input.Options{
		Required: true,
		Loop:     true,
	})
	clientSecret, _ := ui.Ask("Client secret which is registered in the OIDC provider (Default: none):", &input.Options{
		Default:  "",
		Required: false,
	})
	var maxSessionDurationSecondsInt int64
	_, _ = ui.Ask("The max session duration, in seconds, of the role session [900-43200] (Default: 3600):", &input.Options{
		Default:  "3600",
		Required: true,
		Loop:     true,
		ValidateFunc: func(s string) error {
			maxSessionDurationSecondsInt, err := strconv.ParseInt(s, 10, 64)
			if err != nil || maxSessionDurationSecondsInt < 900 || maxSessionDurationSecondsInt > 43200 {
				return fmt.Errorf("input must be 900-43200")
			}
			return nil
		},
	})

	toolConfig, err := readConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file")
	}

	updatedConfig := toolConfig[providerName]
	updatedConfig.OIDCServer = oidcServer
	updatedConfig.AuthURL = authURL
	updatedConfig.TokenURL = tokenURL
	updatedConfig.ClientID = clientID
	updatedConfig.ClientSecret = clientSecret
	updatedConfig.MaxSessionDurationSeconds = maxSessionDurationSecondsInt
	toolConfig[providerName] = updatedConfig

	err = writeConfig(toolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to write config file")
	}

	return updatedConfig, nil
}
