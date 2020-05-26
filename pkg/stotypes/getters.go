package stotypes

func FindDekEnvelope(keyId string, kenvs []KeyEnvelope) *KeyEnvelope {
	for _, kenv := range kenvs {
		if kenv.KeyId == keyId {
			return &kenv
		}
	}

	return nil
}
