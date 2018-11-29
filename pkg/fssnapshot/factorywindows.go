// +build windows

package fssnapshot

func PlatformSpecificSnapshotter() Snapshotter {
	return WindowsSnapshotter()
}
