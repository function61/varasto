package stoutils

import (
	"testing"

	"github.com/function61/gokit/assert"
)

func TestParseDomainSocketPath(t *testing.T) {
	assert.EqualString(t, ParseDomainSocketPath("domainsocket:///var/run/docker.sock"), "/var/run/docker.sock")
	assert.EqualString(t, ParseDomainSocketPath("domainsocket:/var/run/docker.sock"), "")
	assert.EqualString(t, ParseDomainSocketPath(":80"), "")
}
