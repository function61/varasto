// Interface for writing blob store adapters to Varasto
package blobstore

import (
	"context"
	"io"

	"github.com/function61/varasto/pkg/stotypes"
)

type Driver interface {
	// backing store must be idempotent, i.e. writing same blob again must not change outcome.
	// write also must be atomic. Fetch() must not return anything before store is completed succesfully.
	RawStore(ctx context.Context, ref stotypes.BlobRef, content io.Reader) error

	// raw = driver doesn't do any encryption/compression/integrity verifications,
	//       they are done at a higher level.
	// if blob is not found, error must report os.IsNotExist(err) == true
	RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error)

	// if blob is stored in multiple volumes, disk access controller fetches from the volume
	// (that is mounted) with the lowest routing cost.
	// currently 10 for local disks, 20 for cloud services.
	RoutingCost() int
}
