package buptypes

import (
	"encoding/hex"
)

const (
	NoParentId = ""
)

type BlobRef []byte

func BlobRefFromHex(serialized string) (*BlobRef, error) {
	bytes, err := hex.DecodeString(serialized)
	if err != nil {
		return nil, ErrBadBlobRef
	}

	return BlobRefFromBytes(bytes)
}

func BlobRefFromBytes(bytes []byte) (*BlobRef, error) {
	if len(bytes) != 32 {
		return nil, ErrBadBlobRef
	}

	br := BlobRef(bytes)
	return &br, nil
}

func (b *BlobRef) AsHex() string {
	return hex.EncodeToString([]byte(*b))
}

func (b *BlobRef) AsSha256Sum() []byte {
	return []byte(*b)
}

type CreateCollectionRequest struct {
	Name              string `json:"name"`
	ParentDirectoryId string `json:"parent_directory_id"`
}

type VolumeDriverKind string

const (
	VolumeDriverKindLocalFs VolumeDriverKind = "local-fs"
)
