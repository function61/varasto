package stoserver

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"testing"
)

func TestEncryptAndDecryptKek(t *testing.T) {
	ks123 := keyStoreWith(testPrivateKey1, testPrivateKey2, testPrivateKey3)
	ks2 := keyStoreWith(testPrivateKey2)
	ks3 := keyStoreWith(testPrivateKey3)

	kenv, err := ks123.EncryptDek("dummyKeyId", []byte("my secret message"), []string{
		"SHA256:NAgdE9bxrBpJu0S2ehoWW+IE/t+w0pIJ6HvRrgkwuOI", // pubKey1
		"SHA256:LU4ylKik0FBhdx1CUYDJcBpwRGwm85cF+Xz/VesODkA", // pubKey2
	})
	assert.Assert(t, err == nil)

	assert.EqualString(t, kenv.KeyId, "dummyKeyId")
	assert.Assert(t, len(kenv.Slots) == 2)
	assert.EqualString(t, kenv.Slots[0].KekFingerprint, "SHA256:NAgdE9bxrBpJu0S2ehoWW+IE/t+w0pIJ6HvRrgkwuOI")
	assert.EqualString(t, kenv.Slots[1].KekFingerprint, "SHA256:LU4ylKik0FBhdx1CUYDJcBpwRGwm85cF+Xz/VesODkA")

	assert.Assert(t, len(kenv.Slots[0].KeyEncrypted) == 128)
	assert.Assert(t, len(kenv.Slots[1].KeyEncrypted) == 128)

	tryDecryption := func(store *keyStore) string {
		decrypted, err := store.DecryptDek(*kenv)
		if err != nil {
			return err.Error()
		} else {
			return string(decrypted)
		}
	}

	assert.EqualString(t, tryDecryption(ks123), "my secret message")
	assert.EqualString(t, tryDecryption(ks2), "my secret message")
	assert.EqualString(t, tryDecryption(ks3), "don't have any private key to slots of DEK dummyKeyId")

	assert.Assert(t, findDekEnvelope("foo", []stotypes.KeyEnvelope{*kenv}) == nil)
	assert.Assert(t, findDekEnvelope("dummyKeyId", []stotypes.KeyEnvelope{*kenv}) != nil)
}

func TestEncryptEmptyKeyIdOrNoPublicKeys(t *testing.T) {
	ks := keyStoreWith(testPrivateKey1)

	_, err := ks.EncryptDek("", []byte("my secret message"), nil)
	assert.EqualString(t, err.Error(), "empty dekId")

	_, err = ks.EncryptDek("foo", []byte("my secret message"), []string{})
	assert.EqualString(t, err.Error(), "no kekPubKeyFingerprints given")
}

func keyStoreWith(privateKeys ...string) *keyStore {
	ks := newKeyStore()

	for _, privateKey := range privateKeys {
		panicIfError(ks.RegisterPrivateKey(privateKey))
	}

	return ks
}

var (
	testPrivateKey1 = `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQCCgb7Ua7ERt9gyln4op+R6J/DqHoeP3hi0yP7mpcDdW0d1OkdO
fr+x1fCSplZS3CqaEV4RQiswilW+lLhZLpAQ81dMKq1p0udPnlwZWGfV+4cRXHI0
R57uoR9h1hLqMiknis3fyPFP6Rd7OFZpnHV+zOLSMwoXNsQ21vJxVcXRdQIDAQAB
AoGAf3fQdtPUuBST8x0wje8mZvXaBiHZkHiCMxnadldRICOGkQZiHVYJT95BQkt7
JyVp6t+pvDufyaJkC2hhAqJLDP7+efxHlfaa+PduSBClb5/eLQnhTlOYXHGbL3mq
JPhrnFzVTwf5IkNlSNhduvu80heOYBu9qQ9S5ypEMvdWuUUCQQD0mtMB3jNT5baC
O12BklA6eGBD+bVmJrr5cD6LE27ErjfYUHEi1a2PROMRPpJosFQXOLE/at6QSwzn
VtjwX6sDAkEAiJYuELccrwzcovvTe8F2/9wBPt7mtREA3CDsBu1c2sqGBIJaF10j
UpTpd9dIRuRFKd/txqaOjC6VKjRKyITsJwJAEN3IJP3UXjmdvxcm2HNlUtLQGH/U
cUnEZMTHm0FoxukYcrMBShyfzhw66Ap/f/aApeVD25Kb7Ckwp5cGeHSwTwJATyEy
btym8YMyD/p0+y2KE5ER56qbXisLpHwuQZUiRl8uZU5fg0miPSWoXJWMegWlTC0/
Q+cajnwuTtUcvi7D4QJAKCzbJ0KnvxU8ePUxZts+JrVyG0LbPy0GRGPVMVobZ8fP
synk8J2O0CZlhuXHUKCYpANTGk6J06MFeDCBz0sPlA==
-----END RSA PRIVATE KEY-----`
	testPrivateKey2 = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDIVQVgAw5bDA5hPG5IqXMVdbrRqhd10jcwrb5bl5eaqERrQVkd
ws1njow/6yetRllUmeaZZrUefl56oHdIiSoEf10A7GBD5gdedJZzKxrX4mUNyDgr
2FqeXIVgzaZ1wY5DQyCvpd2XO4ANu5tqhcqgIol9aZpPIuY2GQBac1HqtQIDAQAB
AoGAJxPuqHvwIPKJG46eNK5ZNKZyetOjH+iRu30o1NUNTa3lKsbki1mkl77GvPEy
HCrM4iPjR6kxS3F7HJCQtCWNfFqzETQm0PIPLkfyHJ1BevNayV672Kud24Pc5saR
SzKSNrk6vjl7pJC3y9RZ+L6X/CkciPlKKsB+FtOwLpQK7AECQQDq1L2pG0IocpHL
NkYVezWZvfEoAVeJvypM3/7SrI2Hb9F3l6/r4DqO6ZR2ETeoKKyRnlod/1v+BB3S
ZLRKL9ulAkEA2mQjuA4dwhB3nN9Ips2bdWTawMpivF+3T4693ah9dM3WffgHVN2u
cv0SLgjQptRb/qRJoHLRm+Um+ohhMKHl0QJAXoKQcmbOEYlKtAZ73llgESgoznj7
yixt0dK0tAVOUJvoKcGaw8vSxYGshngXdk4oZdLdYgVL+MefWPW+ubzZIQJBAK0l
6AvtZTqPw8XkYb2eFjslEyr3SwD/Al9ZVL+A7rbE2+JT27w1ZjJU4y0MYCFlDOr/
ZkCHyBhJvnWz2xqrwYECQQDXa9Hi1+UgHkHDLMF+tYrD3MvPyqiG1nLcjvcg5cKl
mXKJchGABaSe9g50Ym8REPaavbkXGGQh2tCHOwqjmYOC
-----END RSA PRIVATE KEY-----`
	testPrivateKey3 = `-----BEGIN RSA PRIVATE KEY-----
MIICXQIBAAKBgQDqdbUG7b98RtMpof/kFSslI8QeAfHHfdDjqL9uUBRwGnXUeW3v
0khe/QjNIIuHxkr0dUoLsbRj3mlnXWX7xwK13eA4I4ghxeYdW3CCSfkUj901m9OY
KPQVq9xsujEQHrxBXCVJSP9rkNCB1+6UiB7ytJB4k90kAWixt38kDc1WmwIDAQAB
AoGBAL7yShJwgii2jbc0ZnDdBJxkuo4tyzlLMFqYzf8LXPnHsvruQii0u5gQv6A/
xyM2zUi2VS2c9mr3ciRqnmolNABWZT4IWo71GeN07XDR82keZzJA3MrPafJL503u
cY77ng1Ow6Bfv6Z9iyt2Gcx/5FEK9CqRhJzOVnnUtIPpCssxAkEA/zhBeOg43jzs
MZmd7rw2wUUVsP6lnH8HfqnhrcX8WcOBVsl0FhV13zP8vLyDhRM52H7X1724iXx2
icvJEviBFwJBAOstNDIoKHLRZG6zdTV4YkhreifJz9nWeuaY4PPEe9qggxp8p00G
a4s2qPUlGpY928y5b1zhgc4lMRTTOptLYR0CQF56r9oXdX3n5bQS3yFSsZ5oebg0
/I/rgpXEQ9Q1l86PDmFXYE8QkLsZHrWrv7BSxrY7dqHaDOdwmN04AG6yae8CQA0Z
DWklt2r9onxP3l1GASNLaRhCMyNMwLeLGCw7azJ38hVNj/vIOcEdIDfXAy4O7+jt
AvjHTnVuuNcSFJeFkTkCQQDJVKZtRY3PhjV5/6Nt4yHvGqCGvg7pMJtWiiRZlLLX
BdUJwGOR61xwCgcWNWb2zXvQ1+FIGwqdabeC9jKsT5vk
-----END RSA PRIVATE KEY-----`
)
