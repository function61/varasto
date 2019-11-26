// Client for FUSE server's API
package stofuseclient

import (
	"context"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stofuse/stofusetypes"
)

type Client struct {
	urls *stofusetypes.RestClientUrlBuilder
}

func New(baseUrl string) *Client {
	return &Client{stofusetypes.NewRestClientUrlBuilder(baseUrl)}
}

func (c *Client) UnmountAll(ctx context.Context) error {
	_, err := ezhttp.Post(
		ctx,
		c.urls.FuseUnmountAll())
	return err
}
