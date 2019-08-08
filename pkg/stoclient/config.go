package stoclient

import (
	"fmt"
	"github.com/function61/eventkit/httpcommandclient"
	"github.com/function61/gokit/fileexists"
	"github.com/function61/gokit/jsonfile"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
)

const (
	configFilename = "varastoclient-config.json"
)

type ClientConfig struct {
	ServerAddr    string `json:"server_addr"`
	AuthToken     string `json:"auth_token"`
	FuseMountPath string `json:"fuse_mount_path"`
}

func (c *ClientConfig) ApiPath(path string) string {
	return c.ServerAddr + path
}

func (c *ClientConfig) CommandClient() *httpcommandclient.Client {
	return httpcommandclient.New(c.ApiPath("/command/"), c.AuthToken)
}

func (c *ClientConfig) UrlBuilder() *stoservertypes.RestClientUrlBuilder {
	return stoservertypes.NewRestClientUrlBuilder(c.ServerAddr)
}

func writeConfig(conf *ClientConfig) error {
	confPath, err := configFilePath()
	if err != nil {
		return err
	}

	return jsonfile.Write(confPath, conf)
}

func ReadConfig() (*ClientConfig, error) {
	confPath, err := configFilePath()
	if err != nil {
		return nil, err
	}

	conf := &ClientConfig{}
	return conf, jsonfile.Read(confPath, conf, true)
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
		Use:   "config-init [serverAddr] [authToken]",
		Short: "Initialize configuration, use http://localhost:8066 for dev",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			serverAddr := args[0]
			authToken := args[1]

			confPath, err := configFilePath()
			if err != nil {
				panic(err)
			}

			exists, err := fileexists.Exists(confPath)
			if err != nil {
				panic(err)
			}

			if exists {
				panic("config file already exists")
			}

			conf := &ClientConfig{
				ServerAddr: serverAddr,
				AuthToken:  authToken,
			}

			if err := writeConfig(conf); err != nil {
				panic(err)
			}
		},
	}
}

func configPrintEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "config-print",
		Short: "Prints path to config file & its contents",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			confPath, err := configFilePath()
			if err != nil {
				panic(err)
			}

			fmt.Printf("path: %s\n", confPath)

			exists, err := fileexists.Exists(confPath)
			if err != nil {
				panic(err)
			}

			if !exists {
				fmt.Println(".. does not exist. run config-init to fix that")
				return
			}

			file, err := os.Open(confPath)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			if _, err := io.Copy(os.Stdout, file); err != nil {
				panic(err)
			}
		},
	}
}
