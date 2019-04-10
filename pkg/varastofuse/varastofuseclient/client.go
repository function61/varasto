package varastofuseclient

import (
	"context"
	"github.com/function61/gokit/ezhttp"
)

type Client struct {
	addr string
}

func New() *Client {
	// TODO
	return &Client{"http://192.168.1.103:8689"}
}

func (v *Client) Mount(collectionId string) error {
	_, err := ezhttp.Post(context.Background(), v.addr+"/mounts?collection="+collectionId)
	return err
}

func (v *Client) Unmount(collectionId string) error {
	// FIXME: stupid URL
	_, err := ezhttp.Post(context.Background(), v.addr+"/unmounts?collection="+collectionId)
	return err
}
