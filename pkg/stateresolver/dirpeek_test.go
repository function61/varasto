package stateresolver

import (
	"github.com/function61/gokit/assert"
	"github.com/function61/varasto/pkg/stotypes"
	"strings"
	"testing"
)

func TestDirPeek(t *testing.T) {
	dumpStringSlice := func(sl []string) string {
		return strings.Join(sl, ",")
	}

	dirStructure := []stotypes.File{
		mkFile("foo.txt"),
		mkFile("bar.txt"),
		mkFile("sub/baz.txt"),
		mkFile("sub/subsub1/loooool.png"),
		mkFile("sub/subsub2/hahah.png"),
		mkFile("sub/subsub2/README.md"),
		mkFile("sub/subsub2/inception/going-deeper.mp4"),
		mkFile("not/content/in/a/few/levels.doc"),
	}

	oneCase := func(path string, fileCount int, subDirs string, parentDirs string) {
		peekResult := DirPeek(dirStructure, path)

		assert.Assert(t, len(peekResult.Files) == fileCount)
		assert.EqualString(t, dumpStringSlice(peekResult.SubDirs), subDirs)
		assert.EqualString(t, dumpStringSlice(peekResult.ParentDirs), parentDirs)
	}

	oneCase(".", 2, "sub,not", "")
	oneCase("sub", 1, "sub/subsub1,sub/subsub2", "")
	oneCase("sub/subsub1", 1, "", "sub")
	oneCase("sub/subsub2", 2, "sub/subsub2/inception", "sub")
	oneCase("sub/subsub2/inception", 1, "", "sub/subsub2,sub")
}

func TestParents(t *testing.T) {
	assert.EqualString(
		t,
		strings.Join(parents("sub/subsub2/inception/going-deeper.mp4"), ","),
		"sub/subsub2/inception,sub/subsub2,sub")
}

func TestDirsWithSamePrefix(t *testing.T) {
	// testing bugfix where peeking at "foo" panic'd because it has same prefix as "foobar"
	dirStructure := []stotypes.File{
		mkFile("README.md"),
		mkFile("foo/foo.txt"),
		mkFile("foobar/bar.txt"),
		// also test with above being in a subdir
		mkFile("subdir/foo/foo2.txt"),
		mkFile("subdir/foobar/bar2.txt"),
	}

	assert.EqualString(t, DirPeek(dirStructure, "foo").Files[0].Path, "foo/foo.txt")
	assert.EqualString(t, DirPeek(dirStructure, "foobar").Files[0].Path, "foobar/bar.txt")

	assert.EqualString(t, DirPeek(dirStructure, "subdir/foo").Files[0].Path, "subdir/foo/foo2.txt")
	assert.EqualString(t, DirPeek(dirStructure, "subdir/foobar").Files[0].Path, "subdir/foobar/bar2.txt")
}

func mkFile(path string) stotypes.File {
	return stotypes.File{
		Path: path,
	}
}
