Fyne is used to compile this

Android's file manager APIs are changing
----------------------------------------

https://developer.android.com/training/data-storage/manage-all-files#enable-manage-external-storage-for-testing


AndroidManifest.xml shenanigans
-------------------------------

If one needs to update Fyne arsc.go to get new `AndroidManifest.xml` attributes working for Android's
stupid binary XML:

```
$ docker run --rm -it fyneio/fyne-cross:android-latest-fix bash
$ go get golang.org/x/tools/cmd/stringer
$ go get -d github.com/golang/mobile
$ cd /go/src/github.com/golang/mobile/internal/binres/
$ wget -O /usr/local/android_sdk/platforms/android-15/android.jar https://github.com/Sable/android-platforms/raw/master/android-29/android.jar
$ go generate .
```

