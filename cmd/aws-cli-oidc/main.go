package main

import (
	"log"

	"github.com/docopt/docopt-go"
	"github.com/mbrtargeting/aws-cli-oidc/internal"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	usage := `aws-cli-oidc.

Usage:
  aws-cli-oidc get-cred <idp> <role>
  aws-cli-oidc setup <idp>
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
		err := internal.GetCred(conf.ProviderName, conf.RoleARN)
		if err != nil {
			log.Fatalf("Error during get-cred: %v\n", err)
		}
	} else if conf.Setup {
		err := internal.RunSetup(conf.ProviderName)
		if err != nil {
			log.Fatalf("Error during setup: %v\n", err)
		}
	}
}
