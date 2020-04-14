The recommended way to use Varasto is via **Docker on a Linux PC**. All other combinations
like manual install on Linux, Windows or the likes of Raspberry Pi might work but are
unsupported while Varasto is in beta.

Contents:

- [Limitations of beta version](#limitations-of-beta-version)
- [Linux (Docker)](#linux-docker)
- [Linux (manual)](#linux-manual)
- [Windows](#windows)
- [After Varasto is started](#after-varasto-is-started) (see this after you've installed!)


Limitations of beta version
---------------------------

Varasto is in MVP stage - several important features are not implemented. Such as:

- Security or access controls (only run this in your LAN)
- Anything mentioned in the
  ["General availability" milestone](https://github.com/function61/varasto/milestone/3)
- Also, expect that some settings like replication policies getting rewritten with
  beta-stage updates. Nothing will get rewritten in such a way that it would result in
  losing data.
- Updates between beta versions can be tedious. I will release instructions, but cannot
  promise that they won't contain manual steps like "download backup, replace this from
  backup file, run this command to import backup".


Linux (Docker)
--------------

Find out which version to install from [Docker Hub](https://hub.docker.com/r/fn61/varasto):

```
$ docker run -d --name varasto -p 443:443 fn61/varasto:VERSION
```

NOTE: `-v /dev/disk:/dev/disk:ro --privileged` is required if you want to use SMART or FUSE.
The `/dev/disk` is required for SMART to access the raw block devices (not just the partition
mount point).

NOTE: you'll also have to mount the disks that you plan to use with Varasto. If you just want
to test drive Varasto, you can use `/varasto-db/volume-test/` as your data directory.

Troubleshooting: if you can't access Varasto's web UI, see `$ docker logs varasto`.


### FUSE considerations

For FUSE, add `-v /mnt/stofuse:/mnt/stofuse:shared` to Docker run command. Varasto will then
expose its FS via `/mnt/stofuse/varasto` on your host. The `shared` propagation flag is
required for container's sub-mounts to be visible to the host.

The reason the actual mount is under a directory is, that if you wish to map the mount as
a Samba export via e.g. a Samba container, if we'd map `/mnt/stofuse/varasto` directly,
re-mounting the mountpoint (e.g. FUSE projector restarts) will not get updated to wherever
it's used. tl;dr: we might want to map `/mnt/stofuse` somewhere instead of
`/mnt/stofuse/varasto`.

Pro-tip: for prettier paths, run on your host: `$ ln -s /mnt/stofuse/varasto /varasto`.


Linux (manual)
--------------

Download suitable binary from the Bintray link (in README). Don't worry about `public.tar.gz`
(it's downloaded+extracted automatically if it's missing).

Rename `sto_linux-amd64` -> `sto` and `chmod +x` it.

Make `config.json` in the same directory with content:

```
{	
	"db_location": "varasto.db"
}
```

Now start the server (you may need to use sudo):

```
$ ./sto server
2019/08/02 12:35:04 bootstrap [INFO] generated nodeId: LCb0
2019/08/02 12:35:04 [INFO] node LCb0 (ver. dev) started
```

If everything seems to work, now stop it by pressing `ctrl+c`.

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



Windows
-------

Follow same instructions as for Linux, but there's no autostart yet (the systemd thing),
so you have to just run the .exe file directly from command line.

In the future I think we should research targeting
[Windows Subsystem for Linux](https://en.wikipedia.org/wiki/Windows_Subsystem_for_Linux)
(present since Win10) via Docker to have less moving parts.


After Varasto is started
------------------------

Now you can navigate your browser to `https://localhost/` and **hit "Help" from the menu
to reach the getting started wizard** which will help you set up everything.

(You'll have to approve the "insecure certificate" warning.)

