Linux (manual)
==============

Download suitable binary from the
[newest release](https://github.com/function61/varasto/releases) (you don't need anything
else).

Rename `sto_linux-amd64` -> `sto` and `chmod +x` it.

Make `config.json` in the same directory with content:

```json
{	
	"db_location": "varasto.db"
}
```

Now start the server (you may need to use sudo):

```console
$ ./sto server
2019/08/02 12:35:04 bootstrap [INFO] generated nodeId: LCb0
2019/08/02 12:35:04 [INFO] node LCb0 (ver. dev) started
```

If everything seems to work, now stop it by pressing `ctrl+c`.

Now make it start on system boot (you may need to run this with `sudo`):

```console
$ ./sto server install
Wrote unit file to /etc/systemd/system/varasto.service
Run to enable on boot & to start now:
        $ systemctl enable varasto
        $ systemctl start varasto
        $ systemctl status varasto
```

Just follow above instructions (again, you might need `sudo`).


After Varasto is started
------------------------

Now you can navigate your browser to `https://localhost/` and **hit "Help" from the menu
to reach the getting started wizard** which will help you set up everything.

(You'll have to approve the "insecure certificate" warning.)
