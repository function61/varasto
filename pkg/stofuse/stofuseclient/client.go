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

func (c *Client) Mount(ctx context.Context, collectionId string) error {
	_, err := ezhttp.Post(
		ctx,
		c.urls.FuseMount(),
		ezhttp.SendJson(&stofusetypes.CollectionId{Id: collectionId}))
	return err
}

func (c *Client) Unmount(ctx context.Context, collectionId string) error {
	_, err := ezhttp.Post(
		ctx,
		c.urls.FuseUnmount(),
		ezhttp.SendJson(&stofusetypes.CollectionId{Id: collectionId}))
	return err
}

func (c *Client) UnmountAll(ctx context.Context) error {
	_, err := ezhttp.Post(
		ctx,
		c.urls.FuseUnmountAll())
	return err
}
