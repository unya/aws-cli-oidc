package cmd

import (
	"fmt"
	"github.com/spf13/viper"
)

type OIDCClient struct {
	config     *viper.Viper
}

func InitializeClient(name string) (*OIDCClient, error) {
	config := viper.Sub(name)
	if config == nil {
		fmt.Println("Configuration not found, creating a new one...")
		runSetup()
	}

	client := &OIDCClient{config}

	return client, nil
}
