package cmd

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type OIDCClient struct {
	name   string
	config *providerConfig
}

func InitializeClient(name string) (*OIDCClient, error) {
	configPath := ConfigPath() + "/config.yaml"
	out, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't read config file: %v", err)
	}

	var rootConfig map[string]*providerConfig
	err = yaml.Unmarshal(out, &rootConfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing the config file: %v", err)
	}

	config, exists := rootConfig[name]
	if !exists {
		return nil, fmt.Errorf("configuration not found, run setup to create one")
	}

	client := &OIDCClient{name, config}

	return client, nil
}
