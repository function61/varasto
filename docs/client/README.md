Varasto client
==============

TODO: this doc is very much unfinished.

These instructions mainly apply to desktop platforms. Android and iOS devices have
different mechanisms.


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
