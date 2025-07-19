package stotypes

func FindDekEnvelope(keyID string, kenvs []KeyEnvelope) *KeyEnvelope {
	for _, kenv := range kenvs {
		if kenv.KeyID == keyID {
			return &kenv
		}
	}

	return nil
}
