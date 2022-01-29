package stodupremover

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func scan(removeDuplicates bool, acceptOutdatedDb bool) error {
	db, err := loadDatabase(acceptOutdatedDb)
	if err != nil {
		return err
	}

	actioner := func() Actioner {
		loggerActioner := &LoggerActioner{}

		if removeDuplicates {
			return &TeeActioner{loggerActioner, &RemoveDuplicatesActioner{}}
		} else {
			return loggerActioner
		}
	}()

	if err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		fileContentHashHex, err := hashFileContent(path)
		if err != nil {
			return err
		}

		if duplicateFilename, isDuplicate := db.hashes[fileContentHashHex]; isDuplicate {
			if err := actioner.Duplicate(Item{path}, duplicateFilename); err != nil {
				return err
			}
		} else {
			if err := actioner.NotDuplicate(Item{path}); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return actioner.Finish()
}

func hashFileContent(path string) (string, error) {
	fil, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fil.Close()

	fileContentHash := sha256.New()
	if _, err := io.Copy(fileContentHash, fil); err != nil {
		return "", err
	}

	fileContentHashHex := fmt.Sprintf("%x", fileContentHash.Sum(nil))

	return fileContentHashHex, nil
}
