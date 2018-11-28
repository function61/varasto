package fssnapshot

type Snapshot struct {
	ID                   string
	OriginInSnapshotPath string // path used to access origin in snapshot
	OriginPath           string // snapshot taken from
	SnapshotRootPath     string // path used to access the snapshotted root
}

type Snapshotter interface {
	Snapshot(path string) (*Snapshot, error)
	Release(Snapshot) error
}
