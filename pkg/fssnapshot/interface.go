// Cross-platform filesystem snapshotting library
package fssnapshot

type Snapshot struct {
	ID                    string // opaque platform-specific string (do not use for anything)
	OriginInSnapshotPath  string // path used to access origin in snapshot
	OriginPath            string // snapshot taken from
	SnapshotRootMountPath string // path used to access the snapshotted root
}

type Snapshotter interface {
	Snapshot(path string) (*Snapshot, error)
	Release(Snapshot) error
}
