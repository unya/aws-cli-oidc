package cmd

import (
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	input "github.com/natsukagami/go-input"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "aws-cli-oidc",
	Short: "CLI tool for retrieving AWS temporary credentials using OIDC provider",
	Long:  `CLI tool for retrieving AWS temporary credentials using OIDC provider`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err.Error())
	}
}

var configdir string

const OIDCServer = "oidc_server"
const AuthURL = "auth_url"
const TokenURL = "token_url"
const ClientID = "client_id"
const ClientSecret = "client_secret"
const MaxSessionDurationSeconds = "max_session_duration_seconds"

// OIDC config
const AwsFederationRole = "aws_federation_role"
const AwsFederationRoleSessionName = "aws_federation_role_session_name"

func init() {
	cobra.OnInitialize(initConfig)
}

var ui *input.UI

func initConfig() {
	viper.SetConfigFile(ConfigPath() + "/config.yaml")

	if err := viper.ReadInConfig(); err == nil {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	ui = &input.UI{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}
}

func ConfigPath() string {
	if configdir != "" {
		return configdir
	}
	path := os.Getenv("AWS_CLI_OIDC_CONFIG")
	if path == "" {
		home, err := homedir.Dir()
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		path = home + "/.aws-cli-oidc"
	}
	return path
}

func CheckInstalled(name string) (*OIDCClient, error) {
	return InitializeClient(name)
}
