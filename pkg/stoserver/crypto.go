package stoserver

import (
	"fmt"

	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stokeystore"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

func loadAndFillKeyStore(tx *bbolt.Tx) (*stokeystore.Store, error) {
	keks := []stotypes.KeyEncryptionKey{}
	if err := stodb.KeyEncryptionKeyRepository.Each(stodb.KeyEncryptionKeyAppender(&keks), tx); err != nil {
		return nil, err
	}

	keyStore := stokeystore.New()

	for _, kek := range keks {
		if err := keyStore.RegisterPrivateKey(kek.PrivateKey); err != nil {
			return nil, err
		}
	}

	return keyStore, nil
}

func copyAndReEncryptDekFromAnotherCollection(
	dekID string,
	kekPubKeyFingerprints []string,
	tx *bbolt.Tx,
	ks *stokeystore.Store,
) (*stotypes.KeyEnvelope, error) {
	var newEnvelope *stotypes.KeyEnvelope

	// search for source collections having this encryption key to see if we
	// can decrypt the DEK to inject it into another collection
	if err := stodb.CollectionsByDataEncryptionKeyIndex.Query([]byte(dekID), stodb.StartFromFirst, func(collId []byte) error {
		sourceColl, err := stodb.Read(tx).Collection(string(collId))
		if err != nil {
			return err
		}

		dekEnvelope := stotypes.FindDekEnvelope(dekID, sourceColl.EncryptionKeys)
		if dekEnvelope == nil {
			return fmt.Errorf("(should not happen) encryption key envelope not found coll: %s", collId)
		}

		dek, err := ks.DecryptDek(*dekEnvelope)
		if err != nil {
			// TODO: we should be able to tolerate this, and try if our private key allows
			//       decryption of same DEK from another collection
			return err
		}

		newEnvelope, err = ks.EncryptDek(dekID, dek, kekPubKeyFingerprints)
		if err != nil {
			return err
		}

		return stodb.StopIteration
	}, tx); err != nil {
		return nil, err
	}

	if newEnvelope == nil {
		return nil, fmt.Errorf("no decryptable envelope found for DEK %s", dekID)
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
