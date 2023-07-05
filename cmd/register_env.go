package cmd

import (
	"errors"
	"encoding/json"

	"github.com/spf13/cobra"
)

var registerEnvCmd = &cobra.Command{
	Use:   "register-env",
	Short: "register a new deployment environment",
	Long: `register a new deployment environment so that the Signet broker can be informed of which service versions are deployed to which environments.
	
	flags:

	-n --name           the name of the deployment environment being registered (ex. production)

	-i --ignore-config  ingore .signetrc.yaml file if it exists
	
	-u --broker-url     the scheme, domain, and port where the Signet Broker is being hosted (ex. http://localhost:3000)
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(BrokerBaseURL) == 0 {
			return errors.New("No --broker-url was provided. This is a required flag.")
		}

		if len(name) == 0 {
			return errors.New("No --name was provided. A value for this flag is required.")
		}

		requestBody := EnvBody{name}

		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			return err
		}

		err = RegisterEnvWithBroker(BrokerBaseURL+"/api/environments", jsonData)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(registerEnvCmd)

	registerEnvCmd.Flags().StringVarP(&name, "name", "n", "", "The name of the deployment environment being registered")
}