package stoserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"

	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/function61/varasto/pkg/stoutils"
	"go.etcd.io/bbolt"
)

func (c *cHandlers) KekGenerateOrImport(cmd *stoservertypes.KekGenerateOrImport, ctx *command.Ctx) error {
	data := cmd.Data

	if data == "" {
		var err error
		data, err = generateKek()
		if err != nil {
			return err
		}
	}

	privateKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey([]byte(data))
	if err != nil {
		return err
	}

	fingerprint, err := cryptoutil.Sha256FingerprintForPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	kek := stotypes.KeyEncryptionKey{
		ID:          stoutils.NewKeyEncryptionKeyID(),
		Kind:        "rsa",
		Bits:        privateKey.Size() * 8,
		Created:     ctx.Meta.Timestamp,
		Label:       cmd.Label,
		Fingerprint: fingerprint,
		PublicKey:   string(cryptoutil.MarshalPemBytes(x509.MarshalPKCS1PublicKey(&privateKey.PublicKey), cryptoutil.PemTypeRsaPublicKey)),
		PrivateKey:  string(cryptoutil.MarshalPemBytes(x509.MarshalPKCS1PrivateKey(privateKey), cryptoutil.PemTypeRsaPrivateKey)),
	}

	return c.confreload(c.db.Update(func(tx *bbolt.Tx) error {
		return stodb.KeyEncryptionKeyRepository.Update(&kek, tx)
	}))
}

func generateKek() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}

	return string(cryptoutil.MarshalPemBytes(x509.MarshalPKCS1PrivateKey(privateKey), cryptoutil.PemTypeRsaPrivateKey)), nil
}
