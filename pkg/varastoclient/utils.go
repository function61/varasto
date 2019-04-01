package varastoclient

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func BlobIdxFromOffset(offset uint64) (int, int64) {
	blobIdx := int(offset / chunkSize)
	return blobIdx, int64(offset) - (int64(blobIdx) * chunkSize)
}
