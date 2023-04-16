// `gokit` backports. things that exist in newer version of gokit, but which we cannot update to yet.
package gokitbp

import (
	"time"
)

var (
	DefaultReadHeaderTimeout = 60 * time.Second
)

func Pointer[T any](input T) *T {
	return &input
}

func Must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}

	return value
}
