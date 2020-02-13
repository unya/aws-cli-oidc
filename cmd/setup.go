package cmd

import (
	"fmt"
	input "github.com/natsukagami/go-input"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strconv"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup of aws-cli-oidc",
	Long:  `Interactive setup of aws-cli-oidc. Will prompt you for OIDC provider URL and other settings.`,
	Run:   setup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func setup(cmd *cobra.Command, args []string) {
	runSetup()
}

func runSetup() {
	providerName, _ := ui.Ask("OIDC provider name:", &input.Options{
		Required: true,
		Loop:     true,
	})
	clientID, _ := ui.Ask("Client ID which is registered in the OIDC provider:", &input.Options{
		Required: true,
		Loop:     true,
	})
	clientSecret, _ := ui.Ask("Client secret which is registered in the OIDC provider (Default: none):", &input.Options{
		Default:  "",
		Required: false,
	})
	maxSessionDurationSeconds, _ := ui.Ask("The max session duration, in seconds, of the role session [900-43200] (Default: 3600):", &input.Options{
		Default:  "3600",
		Required: true,
		Loop:     true,
		ValidateFunc: func(s string) error {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil || i < 900 || i > 43200 {
				return errors.New(fmt.Sprintf("Input must be 900-43200"))
			}
			return nil
		},
	})
	awsRole, _ := ui.Ask("AWS federation role (arn:aws:iam::<Account ID>:role/<Role Name>):", &input.Options{
		Required: true,
		Loop:     true,
	})
	awsRoleSessionName, _ := ui.Ask("AWS federation roleSessionName:", &input.Options{
		Required: true,
		Loop:     true,
	})

	config := map[string]string{}

	config[CLIENT_ID] = clientID
	config[CLIENT_SECRET] = clientSecret
	config[MAX_SESSION_DURATION_SECONDS] = maxSessionDurationSeconds
	config[AWS_FEDERATION_ROLE] = awsRole
	config[AWS_FEDERATION_ROLE_SESSION_NAME] = awsRoleSessionName

	viper.Set(providerName, config)

	os.MkdirAll(ConfigPath(), 0700)
	configPath := ConfigPath() + "/config.yaml"
	viper.SetConfigFile(configPath)
	err := viper.WriteConfig()

	if err != nil {
		Writeln("Failed to write %s", configPath)
		Exit(err)
	}

	Writeln("Saved %s", configPath)
}
