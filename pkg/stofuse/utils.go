package stofuse

var reservedInodeCounter = uint64(0)

func nextInode() uint64 {
	reservedInodeCounter++
	return reservedInodeCounter
}
