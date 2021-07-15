package main

import (
	"fmt"
	"log"

	"github.com/docopt/docopt-go"
	"github.com/unya/aws-cli-oidc/internal"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	usage := `aws-cli-oidc.

Usage:
  aws-cli-oidc get-cred <idp> <role> [<seconds>] [print]
  aws-cli-oidc setup <idp>
  aws-cli-oidc cache (show [token]| clear)
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
		Expire       int64  `docopt:"<seconds>"`
		PrintCred    bool   `docopt:"print"`
		Cache        bool   `docopt:"cache"`
		Show         bool   `docopt:"show"`
		Token        bool   `docopt:"token"`
		Clear        bool   `docopt:"clear"`
	}
	if err := arguments.Bind(&conf); err != nil {
		log.Fatalf("%v\n", err)
	}

	switch {
	case conf.GetCred:
		printCred := false
		var expire int64
		expire = 0
		if conf.PrintCred {
			printCred = true
		}
		if conf.Expire > 0 {
			expire = conf.Expire
		}
		err := internal.GetCred(conf.ProviderName, conf.RoleARN, printCred, expire)
		if err != nil {
			log.Fatalf("Error during get-cred: %v\n", err)
		}
	case conf.Setup:
		err := internal.RunSetup(conf.ProviderName)
		if err != nil {
			log.Fatalf("Error during setup: %v\n", err)
		}
	case conf.Cache:
		if conf.Show {
			showToken := false
			if conf.Token {
				showToken = true
			}
			output, err := internal.CacheShow(showToken)
			if err != nil {
				log.Fatalf("Error during cache read: %v\n", err)
			}
			fmt.Print(output)
		} else if conf.Clear {
			if err := internal.CacheClear(); err != nil {
				log.Fatalf("Error during cache clear: %v\n", err)
			}
		}
	}
}
