package fssnapshot

// you can use NullSnapshotter when your application gives the option of using snapshots.
// in the cases where snapshotting is not available (or user doesn't want it), you can do
// your file accessing using the same logic (take snapshot, read files, release snapshot)
// regardless of if snapshotting is actually used or not.

func NullSnapshotter() Snapshotter {
	return &nullSnapshotter{}
}

type nullSnapshotter struct{}

func (l *nullSnapshotter) Snapshot(path string) (*Snapshot, error) {
	return &Snapshot{
		ID:                    "No snapshotting was used",
		OriginPath:            path,
		OriginInSnapshotPath:  path,
		SnapshotRootMountPath: path,
	}, nil
}

func (l *nullSnapshotter) Release(Snapshot) error {
	return nil
}
