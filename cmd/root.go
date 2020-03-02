package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/docopt/docopt-go"
	homedir "github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
)

type OIDCClient struct {
	name   string
	config *providerConfig
}

func Execute() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	usage := `aws-cli-oidc.

Usage:
  aws-cli-oidc get-cred <idp> <role>
  aws-cli-oidc setup
  aws-cli-oidc -h | --help

Options:
  -h --help  Show this screen.`

	arguments, err := docopt.ParseDoc(usage)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	var conf struct {
		GetCred      bool   `docopt:"get-cred"`
		Setup        bool   `docopt:"setup"`
		ProviderName string `docopt:"<idp>"`
		RoleARN      string `docopt:"<role>"`
	}
	if err := arguments.Bind(&conf); err != nil {
		log.Fatalf("%v\n", err)
	}

	if conf.GetCred {
		getCred(conf.ProviderName, conf.RoleARN)
	} else if conf.Setup {
		_, err := runSetup()
		if err != nil {
			log.Fatalf("Error during setup: %v\n", err)
		}
	}
}

var configdir string

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
