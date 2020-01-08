package stotypes

import (
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"time"
)

type Node struct {
	ID           string
	Addr         string
	Name         string
	TlsCert      string
	SmartBackend stoservertypes.SmartBackend
}

type Client struct {
	ID        string
	Created   time.Time
	AuthToken string
	Name      string
}

type ReplicationPolicy struct {
	ID             string
	Name           string
	DesiredVolumes []int
}

type Volume struct {
	ID            int
	UUID          string
	Label         string
	Description   string
	Notes         string
	SerialNumber  string
	Technology    string
	SmartId       string
	SmartReport   string
	Enclosure     string
	EnclosureSlot int // 0 = not defined
	Manufactured  time.Time
	WarrantyEnds  time.Time
	Quota         int64
	BlobSizeTotal int64 // @ compressed & deduplicated
	BlobCount     int64 // does not include volume descriptor blob (sha256=0000..)
}

type VolumeMount struct {
	ID         string
	Volume     int
	Node       string
	Driver     stoservertypes.VolumeDriverKind
	DriverOpts string
}

type Directory struct {
	ID          string
	Parent      string
	Name        string
	Description string
	Type        string
	Metadata    map[string]string
	Sensitivity int // 0(for all eyes) 1(a bit sensitive) 2(for my eyes only)
}

type Collection struct {
	ID             string
	Created        time.Time // earliest of all changesets' file create/update timestamps
	Directory      string
	Name           string
	Description    string
	Sensitivity    int // 0(for all eyes) 1(a bit sensitive) 2(for my eyes only)
	DesiredVolumes []int
	Head           string
	EncryptionKeys []KeyEnvelope // first is for all new blobs, the following for moved/deduplicated ones
	Changesets     []CollectionChangeset
	Metadata       map[string]string
	Tags           []string
}

type CollectionChangeset struct {
	ID           string
	Parent       string
	Created      time.Time
	FilesCreated []File
	FilesUpdated []File
	FilesDeleted []string
}

func (c *CollectionChangeset) AnyChanges() bool {
	return (len(c.FilesCreated) + len(c.FilesUpdated) + len(c.FilesDeleted)) > 0
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
	Ref                       BlobRef
	EncryptionKeyId           string
	Volumes                   []int
	VolumesPendingReplication []int
	Referenced                bool // aborted uploads (ones that do not get referenced by a commit) could leave orphaned blobs
	IsCompressed              bool
	Size                      int32
	SizeOnDisk                int32 // after optional compression
	Crc32                     []byte
}

type IntegrityVerificationJob struct {
	ID                   string
	Started              time.Time
	Completed            time.Time
	VolumeId             int
	LastCompletedBlobRef BlobRef
	BytesScanned         uint64
	ErrorsFound          int
	Report               string
}

type Config struct {
	Key   string
	Value string
}

type KeyEncryptionKey struct {
	ID          string
	Kind        string // rsa | ecdsa
	Bits        int
	Created     time.Time
	Label       string
	Fingerprint string // for public key
	PublicKey   string
	PrivateKey  string
}

type ScheduledJob struct {
	ID          string
	Kind        stoservertypes.ScheduledJobKind
	Description string
	Schedule    string
	Enabled     bool
	NextRun     time.Time
	LastRun     *ScheduledJobLastRun
}

type ScheduledJobLastRun struct {
	Started  time.Time
	Finished time.Time
	Error    string
}

func NewChangeset(
	id string,
	parent string,
	created time.Time,
	filesCreated []File,
	filesUpdated []File,
	filesDeleted []string,
) CollectionChangeset {
	return CollectionChangeset{
		ID:           id,
		Parent:       parent,
		Created:      created,
		FilesCreated: filesCreated,
		FilesUpdated: filesUpdated,
		FilesDeleted: filesDeleted,
	}
}

func NewDirectory(id string, parent string, name string, typ string) *Directory {
	return &Directory{
		ID:       id,
		Parent:   parent,
		Name:     name,
		Metadata: map[string]string{},
		Type:     typ,
	}
}

type KeySlot struct {
	KekFingerprint string `json:"kek_fingerprint"`
	KeyEncrypted   []byte `json:"key_encrypted"`
}

type KeyEnvelope struct {
	KeyId string    `json:"key_id"`
	Slots []KeySlot `json:"slots"`
}
