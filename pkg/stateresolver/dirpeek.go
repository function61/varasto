package stateresolver

import (
	"github.com/function61/gokit/sliceutil"
	"github.com/function61/varasto/pkg/varastotypes"
	"path"
	"strings"
)

type DirPeekResult struct {
	Path       string
	Files      []varastotypes.File
	ParentDirs []string // doesn't include root
	SubDirs    []string
}

// given a bunch of files with paths, we can create a directory model that lets us look
// at one directory at a time, listing its sub- and parent dirs
func DirPeek(files []varastotypes.File, dirToPeek string) *DirPeekResult {
	res := &DirPeekResult{
		Path:       dirToPeek,
		Files:      []varastotypes.File{},
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
		dir := path.Dir(file.Path)

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
func parents(dirPath string) []string {
	ret := []string{}

	curr := path.Dir(dirPath)

	for curr != "." && curr != "/" {
		ret = append(ret, curr)

		curr = path.Dir(curr)
	}

	return ret
}
