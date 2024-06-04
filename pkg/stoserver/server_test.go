package stoserver

import (
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/function61/gokit/cryptoutil"
)

func TestMkWrappedKeypair(t *testing.T) {
	certPem := `-----BEGIN CERTIFICATE-----
MIIBjTCCATOgAwIBAgIQcA8FmTXCBv38IhLOKVOiOzAKBggqhkjOPQQDAjAkMSIw
IAYDVQQKExlFdmVudCBIb3Jpem9uIGludGVybmFsIENBMB4XDTE5MTIxNTE0MDIy
NVoXDTM5MTIxNTE0MDIyNVowJDEiMCAGA1UEChMZRXZlbnQgSG9yaXpvbiBpbnRl
cm5hbCBDQTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABMREdYpF+A54nz0SdYvU
WvR/SBquwmimwRYKHwKOXOt5fqWHfv+2ayvAV/9v8eLQSUdcPGMinqW28ELV981S
O5ujRzBFMA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYB
BQUHAwEwFAYDVR0RBA0wC4IJbG9jYWxob3N0MAoGCCqGSM49BAMCA0gAMEUCIDw1
HpwVUGEkVsDp0Kl556XftcOJcKLkjgeMLERt4TUiAiEAqJZvB40TFLrAShtovcc5
/FwjIqnJX8kT6Pox3QYSspI=
-----END CERTIFICATE-----`

	//nolint:gosec // intentionally insecure key
	keyPem := `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINTwTX5Xt26rkBv44y2dXEwetcT54HZr6v20FBFhW7hboAoGCCqGSM49
AwEHoUQDQgAExER1ikX4DnifPRJ1i9Ra9H9IGq7CaKbBFgofAo5c63l+pYd+/7Zr
K8BX/2/x4tBJR1w8YyKepbbwQtX3zVI7mw==
-----END EC PRIVATE KEY-----`

	cw, err := mkWrappedKeypair([]byte(certPem), []byte(keyPem))
	assert.Assert(t, err == nil)

	assert.EqualString(t, cryptoutil.Identity(cw.cert), "localhost")
	assert.EqualString(t, cryptoutil.Issuer(cw.cert), "Event Horizon internal CA")
}
