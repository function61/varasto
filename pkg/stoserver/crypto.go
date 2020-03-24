package stoserver

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/ssh"
)

type keyStore struct {
	keksPrivate map[string]*rsa.PrivateKey
	keksPublic  map[string]*rsa.PublicKey
}

func newKeyStore() *keyStore {
	return &keyStore{
		keksPrivate: map[string]*rsa.PrivateKey{},
		keksPublic:  map[string]*rsa.PublicKey{},
	}
}

// not safe for concurrent use after boot
func (k *keyStore) RegisterPrivateKey(rsaPrivKeyPemPkcs1 string) error {
	privateKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey([]byte(rsaPrivKeyPemPkcs1))
	if err != nil {
		return err
	}

	fingerprint, err := sha256FingerprintForPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	k.keksPrivate[fingerprint] = privateKey
	k.keksPublic[fingerprint] = &privateKey.PublicKey

	return nil
}

func (k *keyStore) DecryptDek(kenv stotypes.KeyEnvelope) ([]byte, error) {
	for _, slot := range kenv.Slots {
		if privateKey, found := k.keksPrivate[slot.KekFingerprint]; found {
			key, err := privateKey.Decrypt(rand.Reader, slot.KeyEncrypted, &rsa.OAEPOptions{
				Hash: crypto.SHA256,
			})
			if err != nil {
				return nil, err
			}

			return key, nil
		}
	}

	return nil, fmt.Errorf("don't have any private key to slots of DEK %s", kenv.KeyId)
}

func (k *keyStore) EncryptDek(dekId string, dek []byte, kekPubKeyFingerprints []string) (*stotypes.KeyEnvelope, error) {
	if dekId == "" {
		return nil, errors.New("empty dekId")
	}

	if len(kekPubKeyFingerprints) == 0 {
		return nil, errors.New("no kekPubKeyFingerprints given")
	}

	slots := []stotypes.KeySlot{}

	for _, publicKeyFingerprint := range kekPubKeyFingerprints {
		publicKey, found := k.keksPublic[publicKeyFingerprint]
		if !found {
			return nil, fmt.Errorf(
				"request to encrypt with non-registered pubkey: %s",
				publicKeyFingerprint)
		}

		dekCiphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, dek, nil)
		if err != nil {
			return nil, err
		}

		slots = append(slots, stotypes.KeySlot{
			KekFingerprint: publicKeyFingerprint,
			KeyEncrypted:   dekCiphertext,
		})
	}

	return &stotypes.KeyEnvelope{
		KeyId: dekId,
		Slots: slots,
	}, nil
}

func findDekEnvelope(keyId string, kenvs []stotypes.KeyEnvelope) *stotypes.KeyEnvelope {
	for _, kenv := range kenvs {
		if kenv.KeyId == keyId {
			return &kenv
		}
	}

	return nil
}

func sha256FingerprintForPublicKey(publicKey *rsa.PublicKey) (string, error) {
	// need to convert to ssh.PublicKey to be able to use the fingerprint util
	sshPubKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	return ssh.FingerprintSHA256(sshPubKey), nil
}

func copyAndReEncryptDekFromAnotherCollection(
	dekId string,
	kekPubKeyFingerprints []string,
	tx *bbolt.Tx,
	ks *keyStore,
) (*stotypes.KeyEnvelope, error) {
	var newEnvelope *stotypes.KeyEnvelope

	// search for source collections having this encryption key to see if we
	// can decrypt the DEK to inject it into another collection
	if err := stodb.CollectionsByDataEncryptionKeyIndex.Query([]byte(dekId), stodb.StartFromFirst, func(collId []byte) error {
		sourceColl, err := stodb.Read(tx).Collection(string(collId))
		if err != nil {
			return err
		}

		dekEnvelope := findDekEnvelope(dekId, sourceColl.EncryptionKeys)
		if dekEnvelope == nil {
			return fmt.Errorf("(should not happen) encryption key envelope not found coll: %s", collId)
		}

		dek, err := ks.DecryptDek(*dekEnvelope)
		if err != nil {
			// TODO: we should be able to tolerate this, and try if our private key allows
			//       decryption of same DEK from another collection
			return err
		}

		newEnvelope, err = ks.EncryptDek(dekId, dek, kekPubKeyFingerprints)
		if err != nil {
			return err
		}

		return stodb.StopIteration
	}, tx); err != nil {
		return nil, err
	}

	if newEnvelope == nil {
		return nil, fmt.Errorf("no decryptable envelope found for DEK %s", dekId)
	}

	return newEnvelope, nil
}

func extractKekPubKeyFingerprints(coll *stotypes.Collection) []string {
	kekPubKeyFingerprints := []string{}
	for _, slot := range coll.EncryptionKeys[0].Slots {
		kekPubKeyFingerprints = append(kekPubKeyFingerprints, slot.KekFingerprint)
	}

	return kekPubKeyFingerprints
}
