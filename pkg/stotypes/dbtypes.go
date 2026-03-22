package stotypes

import (
	"time"

	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
)

type Node struct {
	ID           string
	Addr         string
	Name         string
	TLSCert      string `msgpack:"TlsCert"`
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
	DesiredVolumes []int // where the policy currently directs data (TODO: rename to CurrentVolumes?)
	MinZones       int   // if >= 2, then data is considered fire etc. disaster safe
}

func (r *ReplicationPolicy) ReplicaCount() int {
	return len(r.DesiredVolumes)
}

type Volume struct {
	ID                 int
	UUID               string
	Label              string
	Description        string
	Notes              string
	SerialNumber       string
	Technology         string
	SmartID            string `msgpack:"SmartId"`
	SmartReport        string
	Zone               string
	Enclosure          string
	EnclosureSlot      int // 0 = not defined
	Manufactured       time.Time
	WarrantyEnds       time.Time
	Quota              int64
	BlobSizeTotal      int64 // @ compressed & deduplicated
	BlobCount          int64 // does not include queued writes or volume descriptor blob (sha256=0000..)
	Decommissioned     *time.Time
	DecommissionReason string
}

type VolumeMount struct {
	ID         string
	Volume     int
	Node       string
	Driver     stoservertypes.VolumeDriverKind
	DriverOpts string
}

type Directory struct {
	ID                string
	Created           time.Time
	MetaCollection    string // backing collection for directory's metadata
	Parent            string
	Name              string
	Type              string
	Sensitivity       int               // 0(for all eyes) 1(a bit sensitive) 2(for my eyes only)
	ReplicationPolicy string            // explicit (for collections it is calculated)
	Deprecated1       map[string]string `msgpack:"Metadata" json:"Metadata"`
	Deprecated2       string            `msgpack:"Description" json:"Description"`
}

type Collection struct {
	ID                string
	Created           time.Time // earliest of all changesets' file create/update timestamps
	Directory         string
	Name              string
	Description       string
	Sensitivity       int           // 0(for all eyes) 1(a bit sensitive) 2(for my eyes only)
	ReplicationPolicy string        // [calculated] effective policy inherited from parent directory
	Head              string        // points to the head changeset. unset only for empty collections
	EncryptionKeys    []KeyEnvelope // first is for all new blobs, the following for moved/deduplicated ones
	Changesets        []CollectionChangeset
	Metadata          map[string]string
	Rating            int // 1-5
	Tags              []string
	GlobalVersion     uint64 `msgpack:"gv"`
}

// this implementation is really bad as a global ordering number (time synchronization
// issues between servers, time jumping back and forth..), but this is temporary until
// we're migrating to EventHorizon which gives us change feeds in a much better way.
func (c *Collection) BumpGlobalVersion() {
	c.GlobalVersion = uint64(time.Now().UTC().UnixNano())
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

func (f *File) CopyEverythingExceptPath(other File) {
	f.Sha256 = other.Sha256
	f.Created = other.Created
	f.Modified = other.Modified
	f.Size = other.Size
	f.BlobRefs = other.BlobRefs
}

type Blob struct {
	Ref                       BlobRef
	EncryptionKeyID           string `msgpack:"EncryptionKeyId"` // DEK that encrypts content of this blob. may be used as hint of collection for which this blob was first uploaded to.
	Volumes                   []int
	VolumesPendingReplication []int  `msgpack:",omitempty"`
	Referenced                bool   `msgpack:",omitempty"` // aborted uploads (ones that do not get referenced by a commit) could leave orphaned blobs
	IsCompressed              bool   `msgpack:",omitempty"` // if you have lots of image/video files, most blobs tend to not be compressible
	Size                      int32  // 32 bits is enough, usually blobs are 4 MB
	SizeOnDisk                int32  // after optional compression. TODO: merge with `IsCompressed`?
	Crc32                     []byte // so we can check file integrity without needing to decrypt (= have access to encryption keys)
}

type IntegrityVerificationJob struct {
	ID                   string
	Started              time.Time
	Completed            time.Time
	SampleSpecification  *string // sample only a subset of items. half: `0`. if you later want to process the second half: `1`. if you want to process quarter: `00`. then the remaining quarters would be (`01`, `10` and `11`). 1 % (can be approximated as 1/128) would be `0000000`.
	VolumeID             int     `msgpack:"VolumeId"`
	LastCompletedBlobRef BlobRef
	BytesScanned         uint64
	BytesSkipped         uint64 // if sampling is in place, we'll be skipping some blobs
	ErrorsFound          int    // in theory this could be derived from report items, but we cannot as the report may be truncated if there's lots of problems
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

func (s ScheduledJobLastRun) Runtime() time.Duration {
	return s.Finished.Sub(s.Started)
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
		ID:      id,
		Created: time.Now(),
		Parent:  parent,
		Name:    name,
		Type:    typ,
	}
}

type KeySlot struct {
	KekFingerprint string `json:"kek_fingerprint"`
	KeyEncrypted   []byte `json:"key_encrypted"`
}

type KeyEnvelope struct {
	KeyID string    `msgpack:"KeyId" json:"key_id"`
	Slots []KeySlot `json:"slots"`
}
