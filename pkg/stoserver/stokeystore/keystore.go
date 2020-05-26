package stokeystore

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"sync"

	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/golang/groupcache/lru"
	"github.com/minio/sha256-simd"
)

type Store struct {
	keksPrivate         map[string]*rsa.PrivateKey
	keksPublic          map[string]*rsa.PublicKey
	decryptedDekCache   *lru.Cache
	decryptedDekCacheMu sync.Mutex
}

func New() *Store {
	return &Store{
		keksPrivate:       map[string]*rsa.PrivateKey{},
		keksPublic:        map[string]*rsa.PublicKey{},
		decryptedDekCache: lru.New(256), // DEK ID => raw encryption key
	}
}

// not safe for concurrent use after boot
func (k *Store) RegisterPrivateKey(rsaPrivKeyPemPkcs1 string) error {
	privateKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPrivateKey([]byte(rsaPrivKeyPemPkcs1))
	if err != nil {
		return err
	}

	fingerprint, err := cryptoutil.Sha256FingerprintForPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	k.keksPrivate[fingerprint] = privateKey
	k.keksPublic[fingerprint] = &privateKey.PublicKey

	return nil
}

func (k *Store) DecryptDek(kenv stotypes.KeyEnvelope) ([]byte, error) {
	k.decryptedDekCacheMu.Lock()
	defer k.decryptedDekCacheMu.Unlock() // lifetime pretty aggressive..

	if cached, found := k.decryptedDekCache.Get(kenv.KeyId); found {
		return cached.([]byte), nil
	}

	for _, slot := range kenv.Slots {
		if privateKey, found := k.keksPrivate[slot.KekFingerprint]; found {
			key, err := privateKey.Decrypt(rand.Reader, slot.KeyEncrypted, &rsa.OAEPOptions{
				Hash: crypto.SHA256,
			})
			if err != nil {
				return nil, err
			}

			k.decryptedDekCache.Add(kenv.KeyId, key)

			return key, nil
		}
	}

	return nil, fmt.Errorf("don't have any private key to slots of DEK %s", kenv.KeyId)
}

func (k *Store) EncryptDek(dekId string, dek []byte, kekPubKeyFingerprints []string) (*stotypes.KeyEnvelope, error) {
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
