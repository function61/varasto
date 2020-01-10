// Writes your blobs to Google Drive
package googledriveblobstore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type googledrive struct {
	varastoDirectoryId string
	logl               *logex.Leveled
	srv                *drive.Service
	reqThrottle        chan interface{}
}

func New(optsSerialized string, logger *log.Logger) (*googledrive, error) {
	opts, err := deserializeConfig(optsSerialized)
	if err != nil {
		return nil, err
	}

	// if modifying scope, delete your cached access token
	config, err := google.ConfigFromJSON([]byte(opts.GoogleCredentialsJson), drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("google.ConfigFromJSON: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	client := getClient(ctx, config)

	gdrive, err := drive.NewService(context.TODO(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("drive.NewService: %v", err)
	}

	return &googledrive{
		varastoDirectoryId: opts.VarastoDirectoryId,
		logl:               logex.Levels(logger),
		srv:                gdrive,
		// default quota seems to be "1 000 queries per 100 seconds per user", so that makes
		// for ten a second
		reqThrottle: mkBurstThrottle(10, 1*time.Second),
	}, nil
}

func (g *googledrive) RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	fileId, err := g.resolveFileIdByRef(ctx, ref)
	if err != nil {
		if err, ok := err.(*googleapi.Error); ok && err.Code == http.StatusNotFound {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	<-g.reqThrottle
	res, err := g.srv.Files.Get(fileId).Context(ctx).Download()
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

func (g *googledrive) RawStore(ctx context.Context, ref stotypes.BlobRef, content io.Reader) error {
	// we've to do this, because Google Drive wouldn't give us an error because it
	// allows >1 files with same filename in same directory
	_, err := g.resolveFileIdByRef(ctx, ref)
	if err == nil {
		// would actually deserve WARN level error, but we don't have that log level
		g.logl.Error.Printf("tried to store a blob that is already present: %s", ref.AsHex())
		return nil // file exists already, so it is technically a success
	}
	if err != os.ErrNotExist {
		return err
	}

	<-g.reqThrottle
	if _, err := g.srv.Files.Create(&drive.File{
		Name:     toGoogleDriveName(ref),
		Parents:  []string{g.varastoDirectoryId},
		MimeType: "application/vnd.varasto.blob",
	}).Media(content).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gdrive Create: %v", err)
	}

	return nil
}

func (g *googledrive) RoutingCost() int {
	return 20
}

func (g *googledrive) resolveFileIdByRef(ctx context.Context, ref stotypes.BlobRef) (string, error) {
	// https://twitter.com/joonas_fi/status/1108008997238595590
	exactFilenameInExactFolderQuery := fmt.Sprintf(
		"name = '%s' and '%s' in parents",
		toGoogleDriveName(ref),
		g.varastoDirectoryId)

	// we're searching with a unique sha256 hash,
	// so we should get exactly one result
	<-g.reqThrottle
	listFilesResponse, err := g.srv.Files.List().PageSize(2).
		Fields("files(id, name)").
		Q(exactFilenameInExactFolderQuery).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("List call failed: %v", err)
	}
	if len(listFilesResponse.Files) == 0 {
		return "", os.ErrNotExist
	}

	return listFilesResponse.Files[0].Id, nil
}

func toGoogleDriveName(ref stotypes.BlobRef) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

type Config struct {
	VarastoDirectoryId    string
	GoogleCredentialsJson string
}

func (c *Config) Serialize() (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}

	asJson, err := json.Marshal(&c)
	if err != nil {
		return "", err
	}

	return string(asJson), nil
}

func (c *Config) validate() error {
	if c.VarastoDirectoryId == "" || c.GoogleCredentialsJson == "" {
		return errors.New("VarastoDirectoryId or GoogleCredentialsJson empty")
	}

	return nil
}

func deserializeConfig(serialized string) (*Config, error) {
	c := &Config{}
	if err := json.Unmarshal([]byte(serialized), c); err != nil {
		return nil, err
	}

	if err := c.validate(); err != nil {
		return nil, err
	}

	return c, nil
}

// https://github.com/golang/go/wiki/RateLimiting
func mkBurstThrottle(burst int, dur time.Duration) chan interface{} {
	ch := make(chan interface{}, burst)
	go func() {
		for range time.Tick(dur) {
			for i := 0; i < burst; i++ {
				ch <- nil
			}
		}
	}()

	return ch
}
