package cmd

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	utils "github.com/signet-framework/signet-cli/utils"
)

var port string
var target string
var providerName string

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "start a signet proxy that automatically generates a consumer contract",
	Long: `start a signet server that acts as a transparent HTTP proxy between a consumer service and a mock or stub of a provider service. Signet proxy records requests and responses, and generates a consumer contract based on those which can be published to the Signet broker.

	flags:

	-o --port           the port that signet proxy should run on

	-t --target         the URL of the running provider stub or mock

	-p --path           the relative path and filename that the consumer contract will be written to

	-n -—name           the canonical name of the consumer service

	-m --provider-name  the canonical name of the provider service that the mock or stub represents

	-i --ignore-config  ingore .signetrc.yaml file if it exists (optional)
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path = viper.GetString("proxy.path")
		port = viper.GetString("proxy.port")
		target = viper.GetString("proxy.target")
		name = viper.GetString("proxy.name")
		providerName = viper.GetString("proxy.provider-name")

		err := validateProxyFlags(path, port, target, name, providerName)
		if err != nil {
			return err
		}

		signetRoot, err := getNpmPkgRoot()
		if err != nil {
			return err
		}
		mbPath := signetRoot + "/node_modules/mountebank"
		configPath := signetRoot + "/config.ejs"
		dataDir := signetRoot + "/mbdata"
		stubsDir := dataDir + "/" + port + "/stubs"

		err = setupMbConfig(port, target, configPath)
		if err != nil {
			return err
		}

		mbCmd := exec.Command("npx", mbPath, "--configfile", configPath, "--datadir", dataDir, "--debug", "--nologfile")
		err = mbCmd.Start()
		if err != nil {
			return errors.New("failed to start mountebank: " + err.Error())
		}

		cmd.Println(colorGreen + "Listening" + colorReset + " - Signet proxy is listening on port " + port + " and will proxy messages for " + target)
		cmd.Println("\nHit Ctl + C to stop")

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for range c {
				cmd.Println("\n\ngenerating consumer contract...")

				err, ok := utils.CreatePact(stubsDir, path, name, providerName)
				if err != nil {
					log.Fatal(err)
				}

				if ok {
					cmd.Println("\n" + colorGreen + "Success" + colorReset + " - Signet proxy wrote the consumer contract to " + path)
				} else {
					cmd.Println("\nInfo - No contract was generated because Signet proxy did not record any interactions")
				}
			}
		}()

		err = mbCmd.Wait()
		if err != nil {
			return errors.New("mountebank exited early: " + err.Error())
		}

		return nil
	},
}

func validateProxyFlags(path, port, target, name, providerName string) error {
	if len(path) == 0 {
		return errors.New("No --path was provided. This is a required flag.")
	}

	if len(port) == 0 {
		return errors.New("No --port was provided. This is a required flag.")
	}

	if len(target) == 0 {
		return errors.New("No --target was provided. This is a required flag.")
	}

	if len(name) == 0 {
		return errors.New("No --name was provided. This is a required flag.")
	}

	if len(providerName) == 0 {
		return errors.New("No --provider-name was provided. This is a required flag.")
	}

	return nil
}

func setupMbConfig(port, target, configPath string) error {
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	proxyConfig := utils.ProxyConfig{
		Port:     portInt,
		Name:     "signet-proxy",
		Protocol: "http",
		Stubs: []utils.MbStub{
			utils.MbStub{
				Responses: []utils.MbResponse{
					utils.MbResponse{
						Proxy: utils.MbProxy{
							To:   target,
							Mode: "proxyOnce",
						},
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(proxyConfig)
	if err != nil {
		return err
	}

	err = osWriteFile(configPath, jsonBytes, rwPermissions)
	if err != nil {
		return errors.New("failed to write mountebank config file: " + err.Error())
	}

	return nil
}

func init() {
	RootCmd.AddCommand(proxyCmd)

	proxyCmd.Flags().StringVarP(&path, "path", "p", "", "the relative path and filename that the consumer contract will be written to")
	proxyCmd.Flags().StringVarP(&port, "port", "o", "", "the port that signet proxy should run on")
	proxyCmd.Flags().StringVarP(&target, "target", "t", "", "the URL of the running provider stub or mock")
	proxyCmd.Flags().StringVarP(&name, "name", "n", "", "the canonical name of the consumer service")
	proxyCmd.Flags().StringVarP(&providerName, "provider-name", "m", "", "the canonical name of the provider service that the mock or stub represents")

	viper.BindPFlag("proxy.path", proxyCmd.Flags().Lookup("path"))
	viper.BindPFlag("proxy.port", proxyCmd.Flags().Lookup("port"))
	viper.BindPFlag("proxy.target", proxyCmd.Flags().Lookup("target"))
	viper.BindPFlag("proxy.name", proxyCmd.Flags().Lookup("name"))
	viper.BindPFlag("proxy.provider-name", proxyCmd.Flags().Lookup("provider-name"))
}
