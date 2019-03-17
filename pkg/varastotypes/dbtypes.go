package varastotypes

import (
	"time"
)

type Node struct {
	ID   string `storm:"id"`
	Addr string
	Name string
}

type Client struct {
	ID        string `storm:"id"`
	AuthToken string
	Name      string
}

type ReplicationPolicy struct {
	ID             string `storm:"id"`
	Name           string
	DesiredVolumes []int
}

type Volume struct {
	ID            int `storm:"id"`
	UUID          string
	Label         string
	Description   string
	Quota         int64
	BlobSizeTotal int64
	BlobCount     int64
}

type VolumeMount struct {
	ID         string `storm:"id"`
	Volume     int
	Node       string
	Driver     VolumeDriverKind
	DriverOpts string
}

type Directory struct {
	ID          string `storm:"id"`
	Parent      string `storm:"index"`
	Name        string
	Description string
	Sensitivity int // 0(for all eyes) 1(a bit sensitive) 2(for my eyes only)
}

type Collection struct {
	ID                string `storm:"id"`
	Directory         string
	Name              string
	Description       string
	ReplicationPolicy string
	Head              string
	Changesets        []CollectionChangeset
}

type CollectionChangeset struct {
	ID           string `storm:"id"`
	Parent       string
	Created      time.Time
	FilesCreated []File
	FilesUpdated []File
	FilesDeleted []string
}

type File struct {
	Path     string
	Sha256   string
	Created  time.Time
	Modified time.Time
	Size     int64
	BlobRefs []string // TODO: use explicit datatype?
}

type Blob struct {
	Ref                       BlobRef `storm:"id"`
	Volumes                   []int
	VolumesPendingReplication []int
	IsPendingReplication      bool `storm:"index"`
	Referenced                bool // aborted uploads (ones that do not get referenced by a commit) could leave orphaned blobs
}
