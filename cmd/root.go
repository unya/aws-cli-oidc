package cmd

import (
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	input "github.com/natsukagami/go-input"
	"github.com/spf13/cobra"
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

func init() {
	cobra.OnInitialize(initConfig)
}

var ui *input.UI

func initConfig() {
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
