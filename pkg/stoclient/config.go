package stoclient

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/function61/eventkit/httpcommandclient"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/spf13/cobra"
)

const (
	configFilename = "varastoclient-config.json"
)

type ClientConfig struct {
	ServerAddr                string `json:"server_addr"` // example: "https://localhost"
	AuthToken                 string `json:"auth_token"`
	FuseMountPath             string `json:"fuse_mount_path"`
	TlsInsecureSkipValidation bool   `json:"tls_insecure_skip_validation"`
}

func (c *ClientConfig) CommandClient() *httpcommandclient.Client {
	return httpcommandclient.New(c.ServerAddr+"/command/", c.AuthToken, c.HttpClient())
}

func (c *ClientConfig) UrlBuilder() *stoservertypes.RestClientUrlBuilder {
	return stoservertypes.NewRestClientUrlBuilder(c.ServerAddr)
}

func (c *ClientConfig) HttpClient() *http.Client {
	client := http.DefaultClient

	if c.TlsInsecureSkipValidation {
		client = ezhttp.InsecureTlsClient
	}

	return client
}

func WriteConfig(conf *ClientConfig) error {
	confPath, err := configFilePath()
	if err != nil {
		return err
	}

	return jsonfile.Write(confPath, conf)
}

func ReadConfig() (*ClientConfig, error) {
	confPath, err := configFilePath()
	if err != nil {
		return nil, fmt.Errorf("Varasto client config: %v", err)
	}

	conf := &ClientConfig{}
	if err := jsonfile.Read(confPath, conf, true); err != nil {
		return nil, fmt.Errorf("Varasto client config: %v", err)
	}

	if strings.HasSuffix(conf.ServerAddr, "/") {
		return nil, fmt.Errorf(
			"Varasto client config: server_addr must not end in '/'; got %s",
			conf.ServerAddr)
	}

	return conf, nil
}

func configFilePath() (string, error) {
	usersHomeDirectory, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(usersHomeDirectory, configFilename), nil
}

func configInitEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "config-init [serverAddr] [authToken] [fuseMountPath]",
		Short: "Initialize configuration, use https://localhost for dev",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			serverAddr := args[0]
			authToken := args[1]
			fuseMountPath := args[2]

			confPath, err := configFilePath()
			exitIfError(err)

			exists, err := fileexists.Exists(confPath)
			exitIfError(err)

			if exists {
				exitIfError(errors.New("config file already exists"))
			}

			conf := &ClientConfig{
				ServerAddr:    serverAddr,
				AuthToken:     authToken,
				FuseMountPath: fuseMountPath,
			}

			exitIfError(WriteConfig(conf))
		},
	}
}

func configPrintEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "config-print",
		Short: "Prints path to config file & its contents",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			confPath, err := ConfigFilePath()
			exitIfError(err)

			fmt.Printf("path: %s\n", confPath)

			exists, err := fileexists.Exists(confPath)
			exitIfError(err)

			if !exists {
				fmt.Println(".. does not exist. run config-init to fix that")
				return
			}

			file, err := os.Open(confPath)
			exitIfError(err)
			defer file.Close()

			_, err = io.Copy(os.Stdout, file)
			exitIfError(err)
		},
	}
}
