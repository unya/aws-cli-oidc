package internal

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type providerConfig struct {
	OIDCServer                string `yaml:"oidc_server"`
	AuthURL                   string `yaml:"auth_url"`
	TokenURL                  string `yaml:"token_url"`
	ClientID                  string `yaml:"client_id"`
	ClientSecret              string `yaml:"client_secret"`
	Region                    string `yaml:"aws_region"`
	MaxSessionDurationSeconds int64  `yaml:"max_session_duration_seconds"`
}

func ConfigPath() string {
	path := os.Getenv("AWS_CLI_OIDC_CONFIG_FILE")
	if path != "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return home + "/.aws-cli-oidc/config.yaml"
}

var configPath = ConfigPath()

func readConfig() (map[string]*providerConfig, error) {
	out, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file")
	}

	var toolConfig map[string]*providerConfig
	err = yaml.Unmarshal(out, &toolConfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing the config file")
	}

	return toolConfig, nil
}

func writeConfig(toolConfig map[string]*providerConfig) error {
	bytes, err := yaml.Marshal(toolConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml config")
	}

	err = ioutil.WriteFile(configPath, bytes, 0700)
	if err != nil {
		return fmt.Errorf("failed to write config file")
	}

	return nil
}

func readProviderConfig(providerName string) (*providerConfig, error) {
	toolConfig, err := readConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file")
	}

	config, exists := toolConfig[providerName]
	if !exists {
		return nil, fmt.Errorf("configuration not found, run setup to create one")
	}

	return config, nil
}
