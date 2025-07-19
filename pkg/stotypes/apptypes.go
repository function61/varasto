package stotypes

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

const (
	BlobSize   = 4 * mebibyte
	mebibyte   = 1024 * 1024
	NoParentID = ""
)

type BlobRef []byte

func BlobRefFromHex(serialized string) (*BlobRef, error) {
	bytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, fmt.Errorf("bad blob ref: %w", err)
	}

	return BlobRefFromBytes(bytes)
}

func BlobRefFromBytes(bytes []byte) (*BlobRef, error) {
	if len(bytes) != 32 {
		return nil, fmt.Errorf("bad blob ref: expecting 32 bytes (got %d)", len(bytes))
	}

	br := BlobRef(bytes)
	return &br, nil
}

func (b *BlobRef) Equal(other BlobRef) bool {
	return bytes.Equal(*b, other)
}

func (b *BlobRef) AsHex() string {
	return hex.EncodeToString([]byte(*b))
}

func (b *BlobRef) AsSha256Sum() []byte {
	return []byte(*b)
}
