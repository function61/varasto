package fssnapshot

import (
	"github.com/function61/gokit/assert"
	"testing"
)

func TestFindSnapshotDeviceFromDetailsOutput(t *testing.T) {
	exampleOutput := `vssadmin 1.1 - Volume Shadow Copy Service administrative command-line tool
(C) Copyright 2001-2013 Microsoft Corp.

Contents of shadow copy set ID: {2caa3819-940a-42ef-a39f-f01f4c75260d}
   Contained 1 shadow copies at creation time: 28/11/2018 15.08.49
      Shadow Copy ID: {984628b9-4972-4af3-8748-e9ec2c810dec}
         Original Volume: (D:)\\?\Volume{10eaffff-0000-0000-0000-602219000000}\
         Shadow Copy Volume: \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy2
         Originating Machine: joonas3
         Service Machine: joonas3
         Provider: 'Microsoft Software Shadow Copy provider 1.0'
         Type: ClientAccessible
         Attributes: Persistent, Client-accessible, No auto release, No writers, Differential

`

	assert.EqualString(
		t,
		findSnapshotDeviceFromDetailsOutput(exampleOutput),
		`\\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy2`)

	assert.EqualString(
		t,
		findSnapshotDeviceFromDetailsOutput("foo"),
		"")
}

func TestFindSnapshotIdFromCreateOutput(t *testing.T) {
	exampleOutput := `Executing (Win32_ShadowCopy)->create()
Method execution successful.
Out Parameters:
instance of __PARAMETERS
{
        ReturnValue = 0;
        ShadowID = "{984628B9-4972-4AF3-8748-E9EC2C810DEC}";
};
`

	assert.EqualString(t,
		findSnapshotIdFromCreateOutput(exampleOutput),
		"{984628B9-4972-4AF3-8748-E9EC2C810DEC}")
}

func TestDriveLetterFromPath(t *testing.T) {
	assert.EqualString(t, driveLetterFromPath("C:/windows"), "C")
	assert.EqualString(t, driveLetterFromPath("D:/games"), "D")
}

func TestOriginPathInSnapshotForWindows(t *testing.T) {
	assert.EqualString(
		t,
		originPathInSnapshot("D:/data/my-cool-origin", "D:/", "D:/snapshots/mysnapshot"),
		"D:/snapshots/mysnapshot/data/my-cool-origin")
}
