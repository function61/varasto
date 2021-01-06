package stodupremover

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/function61/gokit/atomicfilewrite"
	"github.com/function61/gokit/ezhttp"
	"github.com/function61/varasto/pkg/stoclient"
)

type database struct {
	hashes map[string]string
}

func loadDatabase(acceptOldDb bool) (*database, error) {
	dbLocation, err := resolveDbLocation()
	if err != nil {
		return nil, err
	}

	db := &database{
		hashes: map[string]string{},
	}

	f, err := os.Open(dbLocation)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if !acceptOldDb {
		stat, err := f.Stat()
		if err != nil {
			return nil, err
		}

		if time.Since(stat.ModTime()) > 1*time.Hour {
			return nil, errors.New("for your safety refusing to work based on too old duplicate detection DB")
		}
	}

	// ee5903536331ff79d84418db1bfc64b367219511e2da7e982ac6e0be72126124 tortoisehg-2.1.4-hg-1.9.3-x64.msi
	hashAndFilenameParseRe, err := regexp.Compile("^([0-9a-f]{64}) (.+)")
	if err != nil {
		return nil, err
	}

	lines := bufio.NewScanner(f)
	for lines.Scan() {
		line := lines.Text()

		matches := hashAndFilenameParseRe.FindStringSubmatch(line)
		if matches == nil {
			return nil, fmt.Errorf("loadDatabase: failed parsing line: %s", line)
		}

		db.hashes[matches[1]] = matches[2]
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}

	return db, nil
}

func refreshDatabase(ctx context.Context) error {
	dbLocation, err := resolveDbLocation()
	if err != nil {
		return err
	}

	conf, err := stoclient.ReadConfig()
	if err != nil {
		return err
	}

	res, err := ezhttp.Get(
		ctx,
		conf.UrlBuilder().DatabaseExportSha256s(),
		ezhttp.AuthBearer(conf.AuthToken),
		ezhttp.Client(conf.HttpClient()))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return atomicfilewrite.Write(dbLocation, func(sink io.Writer) error {
		_, err := io.Copy(sink, res.Body)
		return err
	})
}

func resolveDbLocation() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, "sto-dupremover.db"), nil
}
