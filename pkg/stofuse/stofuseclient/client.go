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

func (v *Client) Mount(collectionId string) error {
	_, err := ezhttp.Post(
		context.Background(),
		v.urls.FuseMount(),
		ezhttp.SendJson(&stofusetypes.CollectionId{Id: collectionId}))
	return err
}

func (v *Client) Unmount(collectionId string) error {
	_, err := ezhttp.Post(
		context.Background(),
		v.urls.FuseUnmount(),
		ezhttp.SendJson(&stofusetypes.CollectionId{Id: collectionId}))
	return err
}

func (v *Client) UnmountAll() error {
	_, err := ezhttp.Post(
		context.Background(),
		v.urls.FuseUnmountAll())
	return err
}
