package stateresolver

import (
	_ "fmt"
	"github.com/function61/bup/pkg/buptypes"
	"github.com/function61/bup/pkg/sliceutil"
	"path/filepath"
	"strings"
)

type DirPeekResult struct {
	Path       string
	Files      []buptypes.File
	ParentDirs []string // doesn't include root
	SubDirs    []string
}

func DirPeek(files []buptypes.File, dirToPeek string) *DirPeekResult {
	res := &DirPeekResult{
		Path:       dirToPeek,
		Files:      []buptypes.File{},
		ParentDirs: parents(dirToPeek),
		SubDirs:    []string{},
	}

	// "foo" => 1
	// "foo/bar/baz" => 3
	levelOfSubDirToPeek := strings.Count(dirToPeek, "/")

	dirToPeek2 := dirToPeek
	if dirToPeek2 == "." {
		levelOfSubDirToPeek--
		dirToPeek2 = ""
	}

	for _, file := range files {
		// "foo/bar/baz.txt" => "foo/bar"
		dir := filepath.Dir(file.Path)

		if dir == dirToPeek {
			res.Files = append(res.Files, file)
		} else if strings.HasPrefix(dir, dirToPeek2) {
			// "foo/bar" => ["foo", "bar"]
			components := strings.Split(dir, "/")
			if len(components) < levelOfSubDirToPeek+1 {
				continue
			}

			subDir := strings.Join(components[0:levelOfSubDirToPeek+2], "/")

			if !sliceutil.ContainsString(res.SubDirs, subDir) {
				res.SubDirs = append(res.SubDirs, subDir)
			}
		}
	}

	return res
}

// doesn't include root
func parents(path string) []string {
	ret := []string{}

	curr := filepath.Dir(path)

	for curr != "." && curr != "/" {
		ret = append(ret, curr)

		curr = filepath.Dir(curr)
	}

	return ret
}
