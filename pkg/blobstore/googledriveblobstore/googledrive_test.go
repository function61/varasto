package googledriveblobstore

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"golang.org/x/oauth2"
	"testing"
)

func TestToGoogleDriveName(t *testing.T) {
	ref, _ := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")

	assert.EqualString(t, toGoogleDriveName(*ref), "16j7swfXgJRpypq8sAguT41WUeRtPNt2LQLQvzfJ5ZI")
}

func TestSerializeAndDeserializeConfig(t *testing.T) {
	serialized, err := (&Config{
		VarastoDirectoryId: "dummyDirId",
		ClientId:           "dummyClientId",
		ClientSecret:       "dummyClientSecret",
		Token:              &oauth2.Token{},
	}).Serialize()
	assert.Assert(t, err == nil)

	assert.EqualString(t, serialized, `{"directory_id":"dummyDirId","oauth2_client_id":"dummyClientId","oauth2_client_secret":"dummyClientSecret","oauth2_token":{"access_token":"","expiry":"0001-01-01T00:00:00Z"}}`)

	conf, err := deserializeConfig(serialized)
	assert.Assert(t, err == nil)

	assert.EqualString(t, conf.VarastoDirectoryId, "dummyDirId")
	assert.EqualString(t, conf.ClientId, "dummyClientId")
	assert.EqualString(t, conf.ClientSecret, "dummyClientSecret")

	oauth2Conf := Oauth2Config(conf.ClientId, conf.ClientSecret)

	assert.EqualString(t, Oauth2AuthCodeUrl(oauth2Conf), "https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=dummyClientId&redirect_uri=urn%3Aietf%3Awg%3Aoauth%3A2.0%3Aoob&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fdrive&state=state-token")
}
