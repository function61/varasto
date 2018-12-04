Small and crude cross-platform library for taking snapshots of a directory tree.

NOTE: You probably need to run the your binary as admin privileges to be able use snapshots.

Currently supports Windows (via shadow copies) and Linux (via LVM).
Mac support might be possible, but I don't have any plans to do it. PRs are welcome :)


Example code
------------

```
// snapshotDemo("D:/data") // for Windows
// snapshotDemo("/home/macgyver") // for Linux
func snapshotDemo(path string) error {
	snapshotter := fssnapshot.PlatformSpecificSnapshotter(nil)
	snap, err := snapshotter.Snapshot(path)
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

	fmt.Printf("\nOrigin contents\n------------------\n")

	if err := dumpDirectoryContents(snap.OriginPath, os.Stdout); err != nil {
		return err
	}

	fmt.Printf("\n\nSnapshot contents\n------------------\n")

	if err := dumpDirectoryContents(snap.OriginInSnapshotPath, os.Stdout); err != nil {
		return err
	}

	fmt.Printf("\n\n")

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

This should print something like this (on Windows):

```
Took snapshot {108878C4-31CB-475A-B96D-9E08668445C2}. OriginPath<D:/data> OriginInSnapshotPath<D:\snapshots\snap-98e1dbf5\data>
Wrote D:\data\rand-1543500850.txt file in the directory (not in snapshot)

Origin contents
------------------
- foobar.txt
- rand-1543500850.txt

Snapshot contents
------------------
- foobar.txt


Releasing snapshot {108878C4-31CB-475A-B96D-9E08668445C2}
```