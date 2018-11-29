package fssnapshot

// snapshots for Linux's LVM

import (
	"errors"
)

var errLvmNotYetImplemented = errors.New("LVM snapshots not yet implemented")

func LvmSnapshotter() Snapshotter {
	return &lvmSnapshotter{}
}

type lvmSnapshotter struct{}

func (l *lvmSnapshotter) Snapshot(path string) (*Snapshot, error) {
	return nil, errLvmNotYetImplemented
}

func (l *lvmSnapshotter) Release(Snapshot) error {
	return errLvmNotYetImplemented
}
