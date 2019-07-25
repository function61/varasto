package stoserver

import (
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/function61/eventkit/command"
	"github.com/function61/gokit/atomicfilewrite"
	"github.com/function61/gokit/cryptoutil"
	"github.com/function61/gokit/pkencryptedstream"
	"io"
	"strings"
	"time"
)

// TODO: this is Joonas' backup (public, no worries) key - make this configurable
const backupPublicKey = `-----BEGIN RSA PUBLIC KEY-----
MIICCgKCAgEAwUq8WokOMsksCQ2z848d2PC5kXDjMiuOFnsTlMqmrPyuY9nYqix0
5VrNm9sIvvpJJVSDy0wv7EE7gjKgZvJHBkhMKxrXYeYn1XByY2947rl10UUh7+u7
BmCOIYPUeVdPrhb2lBNBj5d+d8avPpOCWrZszbAtL+n6urgW4fXDkHmoThGDucwW
htvQH35UiTARSR9UVEYABL219OhpnA5EcC6TWgaB8t8RoiZL1gfrqqAwz4y36q5e
rmwS5mxHAvF21aeyo8Oyadri7IH9eDL7YUQFwTorQTH0D4Rxzl3FGGqABDNzDLUP
q6bhsJJWHznqQJJ5fue9UW0hLph1y6V+yM90KjVLtEq8DVAK+Ul0KWq62wgM56uL
TILNI51OttttK4SgxegSpijO1rq4Km3dXyEbj7wX0zwkykfzwlzzzPaVUya5Ltmh
Hw+/P+MXOHJJ1Ci3yIBAhdMmyakMZ49DuFBfSEScIvPgldxrkRMiRg9zDqoAgldY
tLkMXxIL1DxtHmCLAXHTCdIt1T8+Nao/SCy4DDz5PvXs4/oNwtrgkOqvkmE1HA7G
AB+yjbXvGA1MptFnjnzLMMb7UCI+vhlpjygU4C+b6ZwYNmPG6tn3pjgZVcVCss40
M8o31e7DQeEXOHL2E1kfUONG4VF2X3EEWPzj0BD/wf0yamkMGgaxeCUCAwEAAQ==
-----END RSA PUBLIC KEY-----
`

func (c *cHandlers) DatabaseBackup(cmd *DatabaseBackup, ctx *command.Ctx) error {
	if c.conf.File.BackupPath == "" {
		return errors.New("BackupPath empty")
	}

	encryptionPublicKey, err := cryptoutil.ParsePemPkcs1EncodedRsaPublicKey(strings.NewReader(backupPublicKey))
	if err != nil {
		return err
	}

	tx, err := c.db.Begin(false)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z07:00") // RFC3339 but time colons replaced with dashes
	filename := fmt.Sprintf(c.conf.File.BackupPath+"/%s.log.gz.aes", ts)

	return atomicfilewrite.Write(filename, func(writer io.Writer) error {
		encryptedStream, err := pkencryptedstream.Writer(writer, encryptionPublicKey)
		if err != nil {
			return err
		}

		compressor := gzip.NewWriter(encryptedStream)

		if err := exportDb(tx, compressor); err != nil {
			return err
		}

		if err := compressor.Close(); err != nil {
			return err
		}

		return encryptedStream.Close()
	})
}
