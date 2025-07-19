package stateresolver

import (
	"path"
	"slices"
	"strings"

	"github.com/function61/varasto/pkg/stotypes"
)

type DirPeekResult struct {
	Path       string
	Files      []stotypes.File
	ParentDirs []string // doesn't include root
	SubDirs    []string
}

// given a bunch of files with paths, we can create a directory model that lets us look
// at one directory at a time, listing its sub- and parent dirs
func DirPeek(files []stotypes.File, dirToPeek string) *DirPeekResult {
	res := &DirPeekResult{
		Path:       dirToPeek,
		Files:      []stotypes.File{},
		ParentDirs: parents(dirToPeek),
		SubDirs:    []string{},
	}

	// "foo" => 1
	// "foo/bar/baz" => 3
	levelOfSubDirToPeek := strings.Count(dirToPeek, "/")

	dirToPeekWithSlash := dirToPeek + "/"
	if dirToPeekWithSlash == "./" {
		levelOfSubDirToPeek--
		dirToPeekWithSlash = ""
	}

	for _, file := range files {
		// "foo/bar/baz.txt" => "foo/bar"
		dir := path.Dir(file.Path)

		if dir == dirToPeek {
			res.Files = append(res.Files, file)
		} else if strings.HasPrefix(dir, dirToPeekWithSlash) {
			// "foo/bar" => ["foo", "bar"]
			components := strings.Split(dir, "/")
			if len(components) < levelOfSubDirToPeek+1 {
				continue
			}

			subDir := strings.Join(components[0:levelOfSubDirToPeek+2], "/")

			if !slices.Contains(res.SubDirs, subDir) {
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
