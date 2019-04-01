package varastofuse

import (
	"context"
	"github.com/function61/varasto/pkg/varastoclient"
	"github.com/function61/varasto/pkg/varastotypes"
	"io/ioutil"
	"log"
	"time"
)

type FsServer struct {
	clientConfig varastoclient.ClientConfig
	blobCache    *BlobCache
}

// FIXME: global
var globalFsServer *FsServer

func NewFsServer(clientConfig varastoclient.ClientConfig) {
	globalFsServer = &FsServer{
		clientConfig: clientConfig,
		blobCache:    NewBlobCache(),
	}
}

const lruCacheSize = 10

type BlobData struct {
	RefHex string
	Data   []byte
	// Loaded bool
}

type BlobCache struct {
	lruCache []*BlobData
}

func NewBlobCache() *BlobCache {
	return &BlobCache{
		lruCache: []*BlobData{},
	}
}

func (b *BlobCache) Get(ctx context.Context, ref varastotypes.BlobRef) (*BlobData, error) {
	refHex := ref.AsHex()

	for _, cachedData := range b.lruCache {
		if cachedData.RefHex == refHex {
			return cachedData, nil
		}
	}

	subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	log.Printf("BlobCache:Get <- %s", refHex)

	blobContent, blobContentCloser, err := varastoclient.DownloadChunk(subCtx, ref, globalFsServer.clientConfig)
	if err != nil {
		return nil, err
	}
	defer blobContentCloser()

	buffered, err := ioutil.ReadAll(blobContent)
	if err != nil {
		return nil, err
	}

	bd := &BlobData{
		RefHex: refHex,
		Data:   buffered,
	}

	if len(b.lruCache) == lruCacheSize {
		// removes oldest item from cache, making room for new
		b.lruCache = b.lruCache[1:lruCacheSize]
	}

	b.lruCache = append(b.lruCache, bd)

	return bd, nil
}
