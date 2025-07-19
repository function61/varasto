// Writes your blobs to Google Drive
package googledriveblobstore

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	oauth2OutOfBandRedirectURLForOfflineApps = "urn:ietf:wg:oauth:2.0:oob" // = user will manually enter code to application
)

type googledrive struct {
	varastoDirectoryID string // ID of directory for storing Varasto blobs
	logl               *logex.Leveled
	srv                *drive.Service
	reqThrottle        chan any
}

func New(optsSerialized string, logger *log.Logger) (*googledrive, error) {
	ctx := context.TODO()

	opts, err := deserializeConfig(optsSerialized)
	if err != nil {
		return nil, err
	}

	client := Oauth2Config(opts.ClientID, opts.ClientSecret).Client(ctx, opts.Token)

	gdrive, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("drive.NewService: %v", err)
	}

	return &googledrive{
		varastoDirectoryID: opts.VarastoDirectoryID,
		logl:               logex.Levels(logger),
		srv:                gdrive,
		// default quota seems to be "1 000 queries per 100 seconds per user", so that makes
		// for ten a second
		reqThrottle: mkBurstThrottle(10, 1*time.Second),
	}, nil
}

func (g *googledrive) RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	fileID, err := g.resolveFileIDByRef(ctx, ref)
	if err != nil {
		if err, ok := err.(*googleapi.Error); ok && err.Code == http.StatusNotFound {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	<-g.reqThrottle
	res, err := g.srv.Files.Get(fileID).Context(ctx).Download()
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

func (g *googledrive) RawStore(ctx context.Context, ref stotypes.BlobRef, content io.Reader) error {
	// we've to do this, because Google Drive wouldn't give us an error because it
	// allows >1 files with same filename in same directory
	_, err := g.resolveFileIDByRef(ctx, ref)
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
		Parents:  []string{g.varastoDirectoryID},
		MimeType: "application/vnd.varasto.blob",
	}).Media(content).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gdrive Create: %v", err)
	}

	return nil
}

func (g *googledrive) RoutingCost() int {
	return 20
}

func (g *googledrive) resolveFileIDByRef(ctx context.Context, ref stotypes.BlobRef) (string, error) {
	// https://twitter.com/joonas_fi/status/1108008997238595590
	exactFilenameInExactFolderQuery := fmt.Sprintf(
		"name = '%s' and '%s' in parents and trashed = false",
		toGoogleDriveName(ref),
		g.varastoDirectoryID)

	<-g.reqThrottle

	// we're searching with a unique sha256 hash,
	// so we should get exactly one result
	listFilesResponse, err := g.srv.Files.List().PageSize(2).
		Fields("files(id, name)").
		Q(exactFilenameInExactFolderQuery).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("op 'List' failed: %w", err)
	}
	if len(listFilesResponse.Files) == 0 {
		return "", os.ErrNotExist
	}

	return listFilesResponse.Files[0].Id, nil
}

func toGoogleDriveName(ref stotypes.BlobRef) string {
	return base64.RawURLEncoding.EncodeToString([]byte(ref))
}

func Oauth2Config(clientID string, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  oauth2OutOfBandRedirectURLForOfflineApps,
		Scopes:       []string{drive.DriveScope},
	}
}

func Oauth2AuthCodeURL(conf *oauth2.Config) string {
	return conf.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

type Config struct {
	VarastoDirectoryID string        `json:"directory_id"`
	ClientID           string        `json:"oauth2_client_id"`
	ClientSecret       string        `json:"oauth2_client_secret"`
	Token              *oauth2.Token `json:"oauth2_token"`
}

func (c *Config) Serialize() (string, error) {
	if err := c.validate(); err != nil {
		return "", err
	}

	asJSON, err := json.Marshal(&c)
	if err != nil {
		return "", err
	}

	return string(asJSON), nil
}

func (c *Config) validate() error {
	if c.ClientID == "" || c.ClientSecret == "" || c.VarastoDirectoryID == "" || c.Token == nil {
		return errors.New("none of the config fields can be empty")
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
func mkBurstThrottle(burst int, dur time.Duration) chan any {
	ch := make(chan any, burst)
	go func() {
		for range time.Tick(dur) {
			for i := 0; i < burst; i++ {
				ch <- nil
			}
		}
	}()

	return ch
}
