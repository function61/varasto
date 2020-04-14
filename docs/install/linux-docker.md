Linux (Docker)
==============

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


After Varasto is started
------------------------

Now you can navigate your browser to `https://localhost/` and **hit "Help" from the menu
to reach the getting started wizard** which will help you set up everything.

(You'll have to approve the "insecure certificate" warning.)
