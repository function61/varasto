package stofuse

import (
	"context"
	"io/ioutil"
	"sync"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/mutexmap"
	"github.com/function61/varasto/pkg/stoclient"
	"github.com/function61/varasto/pkg/stotypes"
)

const lruCacheSize = 10

type BlobData struct {
	Ref  stotypes.BlobRef
	Data []byte
}

type BlobCache struct {
	lruCache     []*BlobData
	lruCacheMu   sync.Mutex
	blobDownload *mutexmap.M
	clientConfig stoclient.ClientConfig
	logl         *logex.Leveled
}

func NewBlobCache(clientConfig stoclient.ClientConfig, logl *logex.Leveled) *BlobCache {
	return &BlobCache{
		lruCache:     []*BlobData{},
		blobDownload: mutexmap.New(),
		clientConfig: clientConfig,
		logl:         logl,
	}
}

func (b *BlobCache) Get(ctx context.Context, ref stotypes.BlobRef, collectionId string) (*BlobData, error) {
	// protect from races ending up in multiple downloads for same blob. we must do this
	// before cache check, because if another thread misses cache, it'll end up re-downloading
	// after first thread fills cache and releases lock
	unlock := b.blobDownload.Lock(ref.AsHex())
	defer unlock()

	if cached := b.getCached(ref); cached != nil {
		return cached, nil
	}

	subCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	b.logl.Debug.Printf("dl %s", ref.AsHex())

	blobContent, blobContentCloser, err := stoclient.DownloadChunk(
		subCtx,
		ref,
		collectionId,
		b.clientConfig)
	if err != nil {
		return nil, err
	}
	defer blobContentCloser()

	buffered, err := ioutil.ReadAll(blobContent)
	if err != nil {
		return nil, err
	}

	bd := &BlobData{
		Ref:  ref,
		Data: buffered,
	}

	b.setCached(bd)

	return bd, nil
}

func (b *BlobCache) getCached(ref stotypes.BlobRef) *BlobData {
	b.lruCacheMu.Lock()
	defer b.lruCacheMu.Unlock()

	for _, cachedData := range b.lruCache {
		if cachedData.Ref.Equal(ref) {
			return cachedData
		}
	}

	return nil
}

func (b *BlobCache) setCached(bd *BlobData) {
	b.lruCacheMu.Lock()
	defer b.lruCacheMu.Unlock()

	if len(b.lruCache) == lruCacheSize {
		// removes oldest item from cache, making room for new
		b.lruCache = b.lruCache[1:lruCacheSize]
	}

	b.lruCache = append(b.lruCache, bd)
}
