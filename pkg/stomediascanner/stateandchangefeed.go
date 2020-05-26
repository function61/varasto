package stomediascanner

import (
	"context"
	"time"

	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

func discoverChanges(ctx context.Context, after string) ([]stoservertypes.CollectionChangefeedItem, error) {
	conf, err := stoclient.ReadConfig()
	if err != nil {
		return nil, err
	}

	changefeedItems := []stoservertypes.CollectionChangefeedItem{}
	if _, err := ezhttp.Get(
		ctx,
		conf.UrlBuilder().CollectionChangefeed(after),
		ezhttp.RespondsJson(&changefeedItems, false),
		ezhttp.Client(conf.HttpClient()),
	); err != nil {
		return nil, err
	}

	return changefeedItems, nil
}

// for reset:
//   $ curl -k -H 'Content-Type: application/json' -d '{"State":""}' https://localhost/command/config.SetThumbnailerState

func discoverState(ctx context.Context, conf *stoclient.ClientConfig) (string, error) {
	return fetchServerConfig(ctx, stoservertypes.CfgMediascannerState, conf)
}

func fetchServerConfig(ctx context.Context, confKey string, conf *stoclient.ClientConfig) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	configValue := stoservertypes.ConfigValue{}
	if _, err := ezhttp.Get(
		ctx,
		conf.UrlBuilder().GetConfig(confKey),
		ezhttp.RespondsJson(&configValue, false),
		ezhttp.Client(conf.HttpClient()),
	); err != nil {
		return "", err
	}

	// can return empty string, but that conveniently works for us, because the changefeed
	// accepts empty string to mean "start from beginning"
	return configValue.Value, nil
}

func setState(ctx context.Context, lastProcessed string, conf *stoclient.ClientConfig) error {
	return conf.CommandClient().Exec(ctx, &stoservertypes.ConfigSetMediascannerState{
		State: lastProcessed,
	})
}
