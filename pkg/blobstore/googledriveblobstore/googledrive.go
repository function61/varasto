// Currently, you need to store a gdrive-credentials.json (gdrive-token.json will be computed)
// in the same directory as you run Varasto from
package googledriveblobstore

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

type googledrive struct {
	varastoDirectoryId string
	logl               *logex.Leveled
	srv                *drive.Service
}

func New(varastoDirectoryId string, logger *log.Logger) (*googledrive, error) {
	gdrive, err := authDance()
	if err != nil {
		return nil, err
	}

	return &googledrive{
		varastoDirectoryId: varastoDirectoryId,
		logl:               logex.Levels(logger),
		srv:                gdrive,
	}, nil
}

func (g *googledrive) RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	fileId, err := g.resolveFileIdByRef(ctx, ref)
	if err != nil {
		return nil, err
	}

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

	if _, err := g.srv.Files.Create(&drive.File{
		Name:     toGoogleDriveName(ref),
		Parents:  []string{g.varastoDirectoryId},
		MimeType: "application/vnd.varasto.blob",
	}).Media(content).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gdrive Create: %v", err)
	}

	return nil
}

func (g *googledrive) Mountable(ctx context.Context) error {
	anyFilesInFolderQuery := fmt.Sprintf("'%s' in parents", g.varastoDirectoryId)

	// just try if a folder query works. it'll 404 if the id is invalid (there seems to
	// be some kind of checksum or something). unfortunately we don't get a failure for
	// non-existing folders (at least deleted one listing succeeded)
	_, err := g.srv.Files.List().PageSize(2).
		Fields("files(id, name)").
		Q(anyFilesInFolderQuery).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("List call failed: %v", err)
	}

	return nil
}

func (g *googledrive) resolveFileIdByRef(ctx context.Context, ref stotypes.BlobRef) (string, error) {
	// https://twitter.com/joonas_fi/status/1108008997238595590
	exactFilenameInExactFolderQuery := fmt.Sprintf(
		"name = '%s' and '%s' in parents",
		toGoogleDriveName(ref),
		g.varastoDirectoryId)

	// we're searching with a unique sha256 hash,
	// so we should get exactly one result
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

func authDance() (*drive.Service, error) {
	b, err := ioutil.ReadFile("gdrive-credentials.json")
	if err != nil {
		return nil, fmt.Errorf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	// config, err := google.ConfigFromJSON(b, drive.DriveMetadataReadonlyScope)
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse client secret file to config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	defer cancel()
	client := getClient(ctx, config)

	srv, err := drive.New(client)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve Drive client: %v", err)
	}

	return srv, nil
}
