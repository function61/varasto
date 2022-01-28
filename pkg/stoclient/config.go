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
	"github.com/function61/gokit/osutil"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/spf13/cobra"
)

type Client struct {
	conf ClientConfig
}

func New(conf ClientConfig) *Client {
	return &Client{conf}
}

func (c *Client) Config() ClientConfig {
	return c.conf
}

type ClientConfig struct {
	ServerAddr                string `json:"server_addr"` // example: "https://localhost"
	AuthToken                 string `json:"auth_token"`
	FuseMountPath             string `json:"fuse_mount_path"`
	TlsInsecureSkipValidation bool   `json:"tls_insecure_skip_validation"`
}

// TODO: this should be temporary
func (c *ClientConfig) Client() *Client {
	return New(*c)
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
	confPath, err := ConfigFilePath()
	if err != nil {
		return err
	}

	return WriteConfigWithPath(conf, confPath)
}

// used by bootstrap
func WriteConfigWithPath(conf *ClientConfig, confPath string) error {
	return jsonfile.Write(confPath, conf)
}

func ReadConfig() (*ClientConfig, error) {
	confPath, err := ConfigFilePath()
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

// returns ~/.config/varasto/varastoclient-config.json
func ConfigFilePath() (string, error) {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("ConfigFilePath: %w", err)
	}

	return filepath.Join(userConfigDir, "varasto", "client-config.json"), nil
}

func configInitEntrypoint() *cobra.Command {
	return &cobra.Command{
		Use:   "config-init [serverAddr] [authToken] [fuseMountPath]",
		Short: "Initialize configuration (helps writing JSON file)",
		Long: `	serverAddr looks like https://localhost
	authToken looks like dTPM59uxWm_uloW4...
	fuseMountPath looks like /mnt/varasto/stofuse OR leave it as empty ("")`,
		Args: cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			osutil.ExitIfError(func(serverAddr string, authToken string, fuseMountPath string) error {
				confPath, err := ConfigFilePath()
				if err != nil {
					return err
				}

				if err := os.MkdirAll(filepath.Dir(confPath), 0700); err != nil {
					return err
				}

				exists, err := fileexists.Exists(confPath)
				if err != nil {
					return err
				}

				if exists {
					return errors.New("config file already exists")
				}

				conf := &ClientConfig{
					ServerAddr:    serverAddr,
					AuthToken:     authToken,
					FuseMountPath: fuseMountPath,
				}

				return WriteConfig(conf)
			}(args[0], args[1], args[2]))
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
			osutil.ExitIfError(err)

			fmt.Printf("file: %s\n", confPath)

			exists, err := fileexists.Exists(confPath)
			osutil.ExitIfError(err)

			if !exists {
				fmt.Printf(".. does not exist. To configure, run:\n    $ %s config-init\n", os.Args[0])
				return
			}

			file, err := os.Open(confPath)
			osutil.ExitIfError(err)
			defer file.Close()

			_, err = io.Copy(os.Stdout, file)
			osutil.ExitIfError(err)
		},
	}
}
