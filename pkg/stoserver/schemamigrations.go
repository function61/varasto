package stoserver

import (
	"fmt"
	"log"

	"github.com/function61/gokit/logex"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"go.etcd.io/bbolt"
)

/*	Schema versions

	v0
	==
	  Changes: no versioning in use
	Migration: n/a

	v1
	==
	  Changes: started schema versioning, added SMART backend type detection to bootstrap
	Migration: change backup header to signature vN, add your desired SMART backend to Node JSON

	v2
	==
	  Changes: added index to DEK to bring back deduplication
	Migration: change backup header to signature vN, import

	v3 (and onwards)
	==
	  Changes: described in function comment
	Migration: automatic
*/

func validateSchemaVersionAndMigrateIfNeeded(db *bbolt.DB, logger *log.Logger) error {
	tx, err := db.Begin(true)
	if err != nil {
		return err
	}
	defer func() { ignoreError(tx.Rollback()) }()

	if err := validateSchemaVersionAndMigrateIfNeededInternal(
		tx,
		logex.Prefix("SchemaMigration", logger),
	); err != nil {
		return err
	}

	return tx.Commit()
}

// returns blorm.ErrBucketNotFound if bootstrap required
func validateSchemaVersionAndMigrateIfNeededInternal(tx *bbolt.Tx, logger *log.Logger) error {
	// the migrations will continue until morale improves
	for {
		schemaVersionInDb, err := stodb.ReadSchemaVersion(tx)
		if err != nil {
			return err
		}

		// no migration needed (or migrations reached a happy level)
		if schemaVersionInDb == stodb.CurrentSchemaVersion {
			return nil
		}

		logex.Levels(logger).Info.Printf(
			"schemaVersionInDb %d, CurrentSchemaVersion %d",
			schemaVersionInDb,
			stodb.CurrentSchemaVersion)

		schemaVersionAfterMigration := schemaVersionInDb + 1

		logex.Levels(logger).Info.Printf(
			"migrating from %d -> %d",
			schemaVersionInDb,
			schemaVersionAfterMigration)

		if err := migrate(schemaVersionInDb, tx); err != nil {
			return err
		}

		if err := stodb.WriteSchemaVersion(schemaVersionAfterMigration, tx); err != nil {
			return err
		}
	}
}

func migrate(schemaVersionInDb uint32, tx *bbolt.Tx) error {
	migrator, found := map[uint32]func(tx *bbolt.Tx) error{
		2: from2to3,
		3: from3to4,
		4: from4to5,
	}[schemaVersionInDb]
	if !found {
		return fmt.Errorf(
			"schema migration %d -> %d not supported",
			schemaVersionInDb,
			schemaVersionInDb+1)
	}

	return migrator(tx)
}

// sets these attributes:
// - directory.ReplicationPolicy
// - coll.ReplicationPolicy
// - volume.Zone
// - replicationPolicy.Zones = 1
func from2to3(tx *bbolt.Tx) error {
	if err := stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)
		coll.ReplicationPolicy = "default"
		return stodb.CollectionRepository.Update(coll, tx)
	}, tx); err != nil {
		return err
	}

	dir, err := stodb.Read(tx).Directory(stoservertypes.RootFolderId)
	if err != nil {
		return err
	}
	dir.ReplicationPolicy = "default"
	if err := stodb.DirectoryRepository.Update(dir, tx); err != nil {
		return err
	}

	if err := stodb.VolumeRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.Volume)
		vol.Zone = "Default"
		return stodb.VolumeRepository.Update(vol, tx)
	}, tx); err != nil {
		return err
	}

	if err := stodb.ReplicationPolicyRepository.Each(func(record interface{}) error {
		vol := record.(*stotypes.ReplicationPolicy)
		vol.MinZones = 1
		return stodb.ReplicationPolicyRepository.Update(vol, tx)
	}, tx); err != nil {
		return err
	}

	return nil
}

// add scheduled task: "update check"
func from3to4(tx *bbolt.Tx) error {
	return stodb.ScheduledJobRepository.Update(stodb.ScheduledJobSeedVersionUpdateCheck(), tx)
}

// - fill GlobalVersion index for changefeed
// - rename IMDB and TMDb metadata keys
func from4to5(tx *bbolt.Tx) error {
	nextGlobalVersion := uint64(1)

	keyStore, err := loadAndFillKeyStore(tx)
	if err != nil {
		return err
	}

	if err := stodb.CollectionRepository.Each(func(record interface{}) error {
		coll := record.(*stotypes.Collection)

		move := func(from string, to string) {
			if value, has := coll.Metadata[from]; has {
				coll.Metadata[to] = value
				delete(coll.Metadata, from)
			}
		}

		move("imdb.id", stoservertypes.MetadataImdbId)
		move("themoviedb.id", stoservertypes.MetadataTheMovieDbMovieId)
		move("themoviedb.episode_id", stoservertypes.MetadataTheMovieDbTvEpisodeId)
		move("themoviedb.tv_id", stoservertypes.MetadataTheMovieDbTvId)

		// deletes are no-op if key does not exist
		delete(coll.Metadata, "thumbnail_url")
		delete(coll.Metadata, "backdrop_url")

		if coll.GlobalVersion == 0 {
			// make sure each collection gets unique version
			nextGlobalVersion++

			coll.GlobalVersion = nextGlobalVersion
		}

		return stodb.CollectionRepository.Update(coll, tx)
	}, tx); err != nil {
		return err
	}

	return stodb.DirectoryRepository.Each(func(record interface{}) error {
		dir := record.(*stotypes.Directory)

		move := func(from string, to string) {
			if value, has := dir.Deprecated1[from]; has {
				dir.Deprecated1[to] = value
				delete(dir.Deprecated1, from)
			}
		}

		delete(dir.Deprecated1, "thumbnail_url")
		delete(dir.Deprecated1, "backdrop_url")

		move("imdb.id", stoservertypes.MetadataImdbId)
		move("themoviedb.id", stoservertypes.MetadataTheMovieDbMovieId)
		move("themoviedb.episode_id", stoservertypes.MetadataTheMovieDbTvEpisodeId)
		move("themoviedb.tv_id", stoservertypes.MetadataTheMovieDbTvId)

		if len(dir.Deprecated1) > 0 {
			metaColl, err := metaCollForDir(dir, tx, keyStore)
			if err != nil {
				return err
			}

			metaColl.Metadata = dir.Deprecated1
			dir.Deprecated1 = nil

			if err := stodb.CollectionRepository.Update(metaColl, tx); err != nil {
				return err
			}
		}

		if dir.Deprecated2 != "" {
			metaColl, err := metaCollForDir(dir, tx, keyStore)
			if err != nil {
				return err
			}

			metaColl.Description = dir.Deprecated2

			if err := stodb.CollectionRepository.Update(metaColl, tx); err != nil {
				return err
			}
		}

		return stodb.DirectoryRepository.Update(dir, tx)
	}, tx)
}
