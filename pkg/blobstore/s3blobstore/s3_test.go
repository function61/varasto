package s3blobstore

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestParseOptionsString(t *testing.T) {
	bucket, regionId, accessKeyId, secret, err := parseOptionsString("varasto-test:eu-central-1:AKIAUZHTE3U35WCD5EHB:wXQJhB...")
	assert.Assert(t, err == nil)

	assert.EqualString(t, bucket, "varasto-test")
	assert.EqualString(t, regionId, "eu-central-1")
	assert.EqualString(t, accessKeyId, "AKIAUZHTE3U35WCD5EHB")
	assert.EqualString(t, secret, "wXQJhB...")
}

func TestParseOptionsStringInvalid(t *testing.T) {
	_, _, _, _, err := parseOptionsString("varasto-test:eu-central-1:AKIAUZHTE3U35WCD5EHB:")
	assert.EqualString(t, err.Error(), "s3 options not in format bucket:region:accessKeyId:secret")
}
