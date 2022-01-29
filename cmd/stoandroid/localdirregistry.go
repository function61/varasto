package main

type LocalDir struct {
	Path                   string
	Label                  string
	enablePush             bool
	enableMover            bool
	enableDiscoverChildren bool
}

func (l LocalDir) Name() string {
	return l.Label
}

func repoDir(path string, label string) LocalDir {
	return LocalDir{
		Path:       path,
		Label:      label,
		enablePush: true,
	}
}

func moverDir(path string, label string) LocalDir {
	return LocalDir{
		Path:                   path,
		Label:                  label,
		enableMover:            true,
		enableDiscoverChildren: true, // mover creates subdirs, which we should enumerate as new LocalDirs
	}
}

var localDirs = []LocalDir{
	repoDir("/storage/emulated/0/data/colornote/backup", "ColorNote backup"),
	moverDir("/storage/emulated/0/DCIM/Camera", "Camera"),
	// repoDir("/storage/emulated/0/DCIM/Camera/2021-01 - Unsorted","2021-01 - Unsorted"),
	// repoDir("/storage/emulated/0/DCIM/Camera/2020-12 - Unsorted","2020-12 - Unsorted"),
	repoDir("/storage/emulated/0/Record/SoundRecord", "Sound recorder"),
	repoDir("/storage/emulated/0/Download", "Downloads"),
	repoDir("/storage/emulated/0/GadgetBridge backup", "GadgetBridge data"),
	repoDir("/storage/emulated/0/Signal backup", "Signal data"),
	// TODO: phone calls
}
