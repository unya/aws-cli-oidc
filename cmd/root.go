package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type OIDCClient struct {
	name   string
	config *providerConfig
}

var rootCmd = &cobra.Command{
	Use:   "aws-cli-oidc",
	Short: "CLI tool for retrieving AWS temporary credentials using OIDC provider",
	Long:  `CLI tool for retrieving AWS temporary credentials using OIDC provider`,
}

var getCredCmd = &cobra.Command{
	Use:   "get-cred <OIDC provider name> <role>",
	Short: "Get AWS credentials and out to stdout",
	Long:  `Get AWS credentials and out to stdout through your OIDC provider authentication.`,
	Run:   getCredCmdRun,
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup of aws-cli-oidc",
	Long:  `Interactive setup of aws-cli-oidc. Will prompt you for OIDC provider URL and other settings.`,
	Run:   setupCmdRun,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err.Error())
	}
}

func setupCmdRun(cmd *cobra.Command, args []string) {
	_, err := runSetup()
	if err != nil {
		log.Fatalf("Error during setup: %v\n", err)
	}
}

func getCredCmdRun(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		log.Fatalln("The OIDC provider name and role ARN is required")
	}
	providerName := args[0]
	roleARN := args[1]

	getCred(providerName, roleARN)
}

var configdir string

func init() {
	rootCmd.AddCommand(getCredCmd)
	rootCmd.AddCommand(setupCmd)
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

	return &OIDCClient{name, config}, nil
}
