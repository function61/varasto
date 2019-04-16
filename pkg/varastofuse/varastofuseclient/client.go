package varastofuseclient

import (
	"context"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/varastofuse/vstofusetypes"
)

type Client struct {
	urls *vstofusetypes.RestClientUrlBuilder
}

func New() *Client {
	// TODO
	return &Client{vstofusetypes.NewRestClientUrlBuilder("http://192.168.1.103:8689")}
}

func (v *Client) Mount(collectionId string) error {
	_, err := ezhttp.Post(
		context.Background(),
		v.urls.FuseMount(),
		ezhttp.SendJson(&vstofusetypes.CollectionId{collectionId}))
	return err
}

func (v *Client) Unmount(collectionId string) error {
	_, err := ezhttp.Post(
		context.Background(),
		v.urls.FuseUnmount(),
		ezhttp.SendJson(&vstofusetypes.CollectionId{collectionId}))
	return err
}
