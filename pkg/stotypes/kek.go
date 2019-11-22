package stotypes

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
)

type KeySlot struct {
	KekFingerprint string `json:"kek_fingerprint"`
	KeyEncrypted   []byte `json:"key_encrypted"`
}

type KeyEnvelope struct {
	KeyId string    `json:"key_id"`
	Slots []KeySlot `json:"slots"`
}

func EncryptEnvelope(keyId string, key []byte, publicKeys []rsa.PublicKey) (*KeyEnvelope, error) {
	if keyId == "" {
		return nil, errors.New("empty keyId")
	}

	if len(publicKeys) == 0 {
		return nil, errors.New("no publicKeys given")
	}

	slots := []KeySlot{}

	for _, publicKey := range publicKeys {
		publicKey := publicKey // pin

		keyEncrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &publicKey, key, nil)
		if err != nil {
			return nil, err
		}

		kekFingerprint, err := Sha256FingerprintForPublicKey(&publicKey)
		if err != nil {
			return nil, err
		}

		slots = append(slots, KeySlot{
			KekFingerprint: kekFingerprint,
			KeyEncrypted:   keyEncrypted,
		})
	}

	return &KeyEnvelope{
		KeyId: keyId,
		Slots: slots,
	}, nil
}

func DecryptKek(kenv KeyEnvelope, privateKey *rsa.PrivateKey) ([]byte, error) {
	kekFingerprint, err := Sha256FingerprintForPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	var slot *KeySlot
	for _, slotCandidate := range kenv.Slots {
		slotCandidate := slotCandidate // pin
		if slotCandidate.KekFingerprint == kekFingerprint {
			slot = &slotCandidate
			break
		}
	}

	if slot == nil {
		return nil, fmt.Errorf("key slot not found for %s", kekFingerprint)
	}

	key, err := privateKey.Decrypt(rand.Reader, slot.KeyEncrypted, &rsa.OAEPOptions{
		Hash: crypto.SHA256,
	})
	if err != nil {
		return nil, err
	}

	return key, nil
}

func FindKeyById(keyId string, kenvs []KeyEnvelope) *KeyEnvelope {
	for _, kenv := range kenvs {
		if kenv.KeyId == keyId {
			return &kenv
		}
	}

	return nil
}

func Sha256FingerprintForPublicKey(publicKey *rsa.PublicKey) (string, error) {
	// need to convert to ssh.PublicKey to be able to use the fingerprint util
	sshPubKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	return ssh.FingerprintSHA256(sshPubKey), nil
}
