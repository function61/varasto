package googledriveblobstore

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stotypes"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func TestToGoogleDriveName(t *testing.T) {
	ref, _ := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")

	assert.EqualString(t, toGoogleDriveName(*ref), "16j7swfXgJRpypq8sAguT41WUeRtPNt2LQLQvzfJ5ZI")
}

func TestSerializeAndDeserializeConfig(t *testing.T) {
	serialized, err := (&Config{
		VarastoDirectoryID: "dummyDirId",
		ClientID:           "dummyClientId",
		ClientSecret:       "dummyClientSecret",
		Token:              &oauth2.Token{},
	}).Serialize()
	assert.Assert(t, err == nil)

	assert.EqualString(t, serialized, `{"directory_id":"dummyDirId","oauth2_client_id":"dummyClientId","oauth2_client_secret":"dummyClientSecret","oauth2_token":{"access_token":"","expiry":"0001-01-01T00:00:00Z"}}`)

	conf, err := deserializeConfig(serialized)
	assert.Assert(t, err == nil)

	assert.EqualString(t, conf.VarastoDirectoryID, "dummyDirId")
	assert.EqualString(t, conf.ClientID, "dummyClientId")
	assert.EqualString(t, conf.ClientSecret, "dummyClientSecret")

	oauth2Conf := Oauth2Config(conf.ClientID, conf.ClientSecret)

	assert.EqualString(t, Oauth2AuthCodeURL(oauth2Conf), "https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=dummyClientId&redirect_uri=urn%3Aietf%3Awg%3Aoauth%3A2.0%3Aoob&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fdrive&state=state-token")
}

func TestRawStoreExistingBlob(t *testing.T) {
	existingContent := []byte("The quick brown fox jumps over the lazy dog")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/files":
			_, _ = io.WriteString(w, `{"files":[{"id":"existing"}]}`)
		case r.URL.Path == "/files/existing" && r.URL.Query().Get("alt") == "media":
			_, _ = w.Write(existingContent)
		default:
			http.Error(w, "unexpected request: "+r.URL.String(), http.StatusNotFound)
		}
	}))
	defer server.Close()

	service, err := drive.NewService(
		context.Background(),
		option.WithEndpoint(server.URL+"/"),
		option.WithHTTPClient(server.Client()),
		option.WithoutAuthentication())
	assert.Ok(t, err)

	throttle := make(chan any, 4)
	for range cap(throttle) {
		throttle <- struct{}{}
	}
	driver := &googledrive{
		varastoDirectoryID: "directory",
		logl:               logex.Levels(logex.Discard),
		srv:                service,
		reqThrottle:        throttle,
	}
	blobRef, err := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")
	assert.Ok(t, err)

	assert.Ok(t, driver.RawStore(context.Background(), *blobRef, bytes.NewReader(existingContent)))

	err = driver.RawStore(context.Background(), *blobRef, bytes.NewReader([]byte("different content")))
	assert.EqualString(t, err.Error(), "existing blob content mismatch: "+blobRef.AsHex())
}
