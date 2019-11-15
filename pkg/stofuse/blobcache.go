package stofuse

import (
	"context"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
	"io/ioutil"
	"time"
)

type FsServer struct {
	clientConfig stoclient.ClientConfig
	blobCache    *BlobCache
	logl         *logex.Leveled
}

// FIXME: global
var globalFsServer *FsServer

func NewFsServer(clientConfig stoclient.ClientConfig, logl *logex.Leveled) {
	globalFsServer = &FsServer{
		clientConfig: clientConfig,
		blobCache:    NewBlobCache(),
		logl:         logl,
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

func (b *BlobCache) Get(ctx context.Context, ref stotypes.BlobRef, collectionId string) (*BlobData, error) {
	refHex := ref.AsHex()

	for _, cachedData := range b.lruCache {
		if cachedData.RefHex == refHex {
			return cachedData, nil
		}
	}

	subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	globalFsServer.logl.Debug.Printf("dl %s", refHex)

	blobContent, blobContentCloser, err := stoclient.DownloadChunk(
		subCtx,
		ref,
		collectionId,
		globalFsServer.clientConfig)
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
