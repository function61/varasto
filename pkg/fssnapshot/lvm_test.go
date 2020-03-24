// +build !windows
// must exclude from Windows build due to syscall.Mount(), syscall.Unmount()

package fssnapshot

import (
	"testing"

	"github.com/function61/gokit/assert"
	"github.com/prometheus/procfs"
)

func TestOriginPathInSnapshot(t *testing.T) {
	sp := "/mnt/snap1"

	assert.EqualString(t, originPathInSnapshot("/home/vagrant/snaptest", "/", sp), "/mnt/snap1/home/vagrant/snaptest")
	assert.EqualString(t, originPathInSnapshot("/home/vagrant/snaptest", "/home", sp), "/mnt/snap1/vagrant/snaptest")
	assert.EqualString(t, originPathInSnapshot("/home/vagrant/snaptest", "/home/vagrant", sp), "/mnt/snap1/snaptest")
	assert.EqualString(t, originPathInSnapshot("/home/vagrant/snaptest", "/home/vagrant/snaptest", sp), "/mnt/snap1")
}

func TestMountForPath(t *testing.T) {
	mounts := []*procfs.Mount{
		{Mount: "/home"},
		{Mount: "/"},
		{Mount: "/var/logs"},
	}

	assert.EqualString(t, mountForPath("/home/vagrant", mounts).Mount, "/home")
	assert.EqualString(t, mountForPath("/home", mounts).Mount, "/home")
	assert.EqualString(t, mountForPath("/root/.ssh/authorized_keys", mounts).Mount, "/")
	assert.EqualString(t, mountForPath("/var/logs/httpd/access.log", mounts).Mount, "/var/logs")
	assert.Assert(t, mountForPath("x", mounts) == nil)
}

func TestDevicePathFromLvsOutput(t *testing.T) {
	output := []byte(`  root   /dev/vagrant-vg/root
  snap1  /dev/vagrant-vg/snap1
  swap_1 /dev/vagrant-vg/swap_1
`)

	assert.EqualString(t, devicePathFromLvsOutput("root", output), "/dev/vagrant-vg/root")
	assert.EqualString(t, devicePathFromLvsOutput("snap1", output), "/dev/vagrant-vg/snap1")
	assert.EqualString(t, devicePathFromLvsOutput("swap_1", output), "/dev/vagrant-vg/swap_1")
	assert.EqualString(t, devicePathFromLvsOutput("notfound", output), "")
}
