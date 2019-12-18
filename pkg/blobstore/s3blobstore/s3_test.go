package s3blobstore

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"testing"
)

func TestDeserializeConfig(t *testing.T) {
	// serialize + deserialize to cover both directions
	conf, err := deserializeConfig((&Config{
		Bucket:          "varasto-test",
		Prefix:          "/",
		AccessKeyId:     "AKIAUZHTE3U35WCD5EHB",
		AccessKeySecret: "wXQJhB...",
		RegionId:        "eu-central-1",
	}).Serialize())
	assert.Assert(t, err == nil)

	assert.EqualString(t, conf.Bucket, "varasto-test")
	assert.EqualString(t, conf.Prefix, "/")
	assert.EqualString(t, conf.AccessKeyId, "AKIAUZHTE3U35WCD5EHB")
	assert.EqualString(t, conf.AccessKeySecret, "wXQJhB...")
	assert.EqualString(t, conf.RegionId, "eu-central-1")
}

func TestDeserializeConfigInvalid(t *testing.T) {
	_, err := deserializeConfig("varasto-test:/:AKIAUZHTE3U35WCD5EHB.missingSecret:eu-central-1")
	assert.EqualString(t, err.Error(), "s3 options not in format bucket:prefix:accessKeyId:secret:region")
}

func TestBlobNamer(t *testing.T) {
	namer := s3BlobNamer{"/mypath/"}

	ref, _ := stotypes.BlobRefFromHex("d7a8fbb307d7809469ca9abcb0082e4f8d5651e46d3cdb762d02d0bf37c9e592")

	name := namer.Ref(*ref)

	assert.EqualString(t, *name, "/mypath/16j7swfXgJRpypq8sAguT41WUeRtPNt2LQLQvzfJ5ZI")
}
