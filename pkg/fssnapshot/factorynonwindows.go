// +build !windows

package fssnapshot

func PlatformSpecificSnapshotter() Snapshotter {
	return LvmSnapshotter("1GB")
}
