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


Download Varasto client binary
------------------------------

You'll find the client binary from your Varasto server's UI.


Place "sto" binary in PATH
--------------------------

Store client app in your `PATH`. This makes it so that you can run `$ sto` from anywhere.


First-time configuration
------------------------

Run:

```
$ sto config-print
```

The command probably tells you that you haven't configured it yet. To configure, run:

```
$ sto config-init
```

The above command gives you instructions on how to proceed.


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
