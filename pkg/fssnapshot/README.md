Beginnings of a cross-platform filesystem snapshotting library.

Currently only supports Windows. LVM support for Linux might come later.

Example code
------------

```
func example() error {
	// TODO: use Go compilation magic for the platform to be bound at compile time
	snapshotter := fssnapshot.WindowsSnapshotter()
	snap, err := snapshotter.Snapshot("D:/data")
	if err != nil {
		return err
	}

	defer func() { // make sure snapshot is deleted in all cases
		fmt.Printf("Releasing snapshot %s\n", snap.ID)

		if err := snapshotter.Release(*snap); err != nil {
			fmt.Printf("failed to release snapshot: %s\n", err.Error())
		}
	}()

	fmt.Printf(
		"Took snapshot %s. OriginPath<%s> OriginInSnapshotPath<%s>\n",
		snap.ID,
		snap.OriginPath,
		snap.OriginInSnapshotPath)

	// store a random file in the origin dir, so we can compare snapshotted and original directory.
	// this created-after-snapshotting file should only be visible in the origin.
	randomFile := filepath.Join(snap.OriginPath, fmt.Sprintf("rand-%d.txt", time.Now().Unix()))

	if err := ioutil.WriteFile(randomFile, nil, 0755); err != nil {
		return fmt.Errorf("error writing file: %s\n", err.Error())
	}

	fmt.Printf("Wrote %s file in the directory (not in snapshot)\n", randomFile)

	fmt.Printf("OriginPath contents\n------------------\n\n")

	if err := dumpDirectoryContents(snap.OriginPath, os.Stdout); err != nil {
		return err
	}

	fmt.Printf("Snapshot OriginInSnapshotPath contents\n------------------\n\n")

	if err := dumpDirectoryContents(snap.OriginInSnapshotPath, os.Stdout); err != nil {
		return err
	}

	return nil
}

func dumpDirectoryContents(path string, output io.Writer) error {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fmt.Fprintf(output, "- %s\n", entry.Name())
	}

	return nil
}
```
