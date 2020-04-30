TODO: this doc is very much unfinished.

These instructions mainly apply to desktop platforms. Android and iOS devices have
different mechanisms. There are also
[more user friendly mechanisms](../../data-interfaces/)
for accessing data.


OS support
----------

| Component      | Linux | Windows | macOS | Android | iOS |
|----------------|-------|---------|-------|---------|-----|
| Varasto client | ☑   | ☑      | ☑   | Soon    | ☐   |


Setting up
----------

### Download

You'll find the client binary from your Varasto server's UI.


### Installation

=== "Linux/macOS"
	Rename the binary to `sto` and give it executable permissions.

	```console
	$ mv sto_linux-amd64 sto
	$ chmod +x sto
	```

	Place the binary in your PATH.

=== "Windows"
	Place the `sto.exe` in your PATH.

	If you don't know where to put it, put it in `C:\Windows\sto.exe`

This makes it so that you can run `$ sto` from anywhere.


### Configuration

If you haven't configured Varasto client yet, `config-print` will give you instructions to fix it:

```console
$ sto config-print
file: /home/joonas/varastoclient-config.json
.. does not exist. To configure, run:
    $ sto config-init
```

Running `$ sto config-init` without any arguments will give you instructions.


How does the cloning interface look like?
-----------------------------------------

!!! note
	Currently the cloning interface is only usable from the command line (i.e. nerds only).
	This will be change to a more user friendly GUI in the future and the pushes will be
	on automatic scheduler to make it an actually usable backup method.

<iframe width="688" height="387" src="https://www.youtube.com/embed/7oPV16_rxKQ" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

Commands (equivalent) from the video:

```console
$ sto clone gu5Yyto9OWE
$ cd "Ender 3 disk"
$ echo "test file content" > Testing.txt
$ sto st
+ Testing.txt

$ sto push
```
