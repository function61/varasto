package stateresolver

import (
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/gokit/assert"
	"strings"
	"testing"
)

func TestDirPeek(t *testing.T) {
	f := func(path string) buptypes.File {
		return buptypes.File{
			Path: path,
		}
	}

	dumpStringSlice := func(sl []string) string {
		return strings.Join(sl, ",")
	}

	dirStructure := []buptypes.File{
		f("foo.txt"),
		f("bar.txt"),
		f("sub/baz.txt"),
		f("sub/subsub1/loooool.png"),
		f("sub/subsub2/hahah.png"),
		f("sub/subsub2/README.md"),
		f("sub/subsub2/inception/going-deeper.mp4"),
		f("not/content/in/a/few/levels.doc"),
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
