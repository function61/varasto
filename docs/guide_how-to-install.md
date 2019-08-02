How to install
==============

NOTE: the installation will become much simpler in the future!


Linux (Docker)
--------------

This will become the primary installation method, but it's not finished yet.


Linux
-----

Download suitable binary from the Bintray link (in README).

Rename `sto_linux-amd64` -> `sto` and `chmod +x` it.

Download & extract `public.tar.gz` to the same directory as you're running the binary from.

Make `config.json` in the same directory with content:

```
{	
	"db_location": "varasto.db",
	"allow_bootstrap": true
}
```

Now start the server:

```
$ ./sto server
2019/08/02 12:35:04 bootstrap [INFO] generated nodeId: LCb0
2019/08/02 12:35:04 [INFO] node LCb0 (ver. dev) started
```

Now stop it by pressing `ctrl+c`. Change `allow_bootstrap` to `false` in the config,
otherwise you'll get this warning the next time you'd start the server:

```
2019/08/02 12:35:52 [ERROR] AllowBootstrap true after bootstrap already done => dangerous
```

Now make it start on system boot (you may need to run this with `sudo`):

```
$ ./sto server install
Wrote unit file to /etc/systemd/system/varasto.service
Run to enable on boot & to start now:
        $ systemctl enable varasto
        $ systemctl start varasto
        $ systemctl status varasto
```

Just follow above instructions (again, you might need `sudo`).

Now you can navigate your browser to `http://localhost:8066/`


Windows
-------

Follow same instructions as for Linux, but there's no autostart yet (the systemd thing),
so you have to just run the .exe file directly from command line.
