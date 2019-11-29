Guide: Network folders
======================

Contents:

- [Motivation](#motivation)
- [OS limitations](#os-limitations)
- [How does it work?](#how-does-it-work-)
- [Architecture](#architecture)
- [Setting up FUSE projector on the same computer as Varasto server](#setting-up-fuse-projector-on-the-same-computer-as-varasto-server)
- [Setting up FUSE projector on a different computer than Varasto server](#setting-up-fuse-projector-on-a-different-computer-than-varasto-server)
- [Testing that FUSE projector is working](#testing-that-fuse-projector-is-working)
- [Setting up Samba](#setting-up-samba)


Motivation
----------

Varasto's web UI is the good for quickly fetching a PDF file, looking at photos, watching
movies/videos or listening to music. But there are times when you just need to interact
with raw files.

For this use case, you could clone a collection to your computer and work with the files.
Cloning has one limitation: you need to download the files first. What if you just need
to access the raw files, but you'd prefer to stream them without downloading them to your
computer? That's what network folders are for!

With Network folders, you can access/stream collections in Varasto as network folders and
files from any device - be it Windows, Linux or Mac computer or a mobile device.


OS limitations
--------------

Currently you need Linux to share data from Varasto as network folders.

Your Varasto server can run on Windows, your Varasto users can run Windows, but you need
Linux to do the network folder exporting.


How does it work?
-----------------

Varasto uses Linux-specific technology called
[FUSE](https://en.wikipedia.org/wiki/Filesystem_in_Userspace) to project content in Varasto
as a regular directory hierarchy that can be accessed from the filesystem.

This directory hierarchy is only accessible on the computer the Varasto FUSE projector runs on.

We can use Samba on Linux to export that directory hierarchy as network share.


Architecture
------------

Taking look at Varasto architecture overview:

![](architecture.png)

The FUSE interface is built on top of the Varasto client library, which talks to the
Varasto server over HTTP. That means that if you want, you can run the FUSE projector
on a different computer than where the Varasto server runs.

Focusing on network folders: Varasto FUSE projector + FUSE + Samba - here's how it could
look like with with separate computers for Varasto server and Varasto FUSE projector:

![](guide_network-folders_architecture.png)

Even if you run them on the same computer, this diagram should explain things pretty good.


Setting up FUSE projector on the same computer as Varasto server
----------------------------------------------------------------

Varasto can manage starting FUSE projector as a subsystem for you. Just go to settings,
and make sure the FUSE projector subsystem is started.

This feature is not yet completely ready - FUSE projector will crash if the client config
file is not present. Read the "different computer than Varasto server" section for details,
and you should read it anyway .

If you're running Docker, you need to bind mount the Varasto FUSE mountpoint from host to
your Varasto server container so the Samba process can benefit from the FUSE mount.


Setting up FUSE projector on a different computer than Varasto server
---------------------------------------------------------------------

This section and process needs cleaning up to make it easier for all users.

Currently I only have a few short pointers for more technical users:

All these steps are done on the computer where you run Varasto FUSE projector on, though it
probably is the same computer where you're running the Varasto server on.

You need to decide where you want the Varasto filesystem mountpoint be placed. In the above
drawing it's `/mnt/varasto` but you can use anything you want. Run `$ mkdir /mnt/varasto`.

Since Varasto FUSE projector is a client, we need to configure how to connect to the server.

Your config should look like this: (`$ sto ...` commands unless prefixed with `server ...`
are Varasto client commands since you'll use the client more often)

```
$ sto config-print
path: /home/joonas/varastoclient-config.json
{
	"server_addr": "http://localhost:8066",
	"auth_token": "qZKcP...",
	"fuse_mount_path": "/mnt/varasto"
}
```

You can run `$ sto config-init` to generate the config file.

For `auth_token` you need to create an API token in `Settings > Users`.

You need to run the projector manually by running `$ sto fuse serve`. TODO: systemd unit
for automatically running it.


Testing that FUSE projector is working
--------------------------------------

Once you've started the FUSE projector, we should test that it works before moving on to
configure Samba. `$ cd /mnt/varasto/id` and under that directory run
`$ cd "$id_of_any_collection_you_have_created" && ls`. FUSE projector dynamically fetches
the latest revision of that collection. You should see its files now.

Once you've got the FUSE projector working, it's time to set up Samba.


Setting up Samba
----------------

Varasto doesn't bundle a Samba server because you might already be running a Samba server
configured to your liking. And if you're using Varasto on Docker, it really goes against
containerization philosophy to run two different products inside one container.

If you already have a Samba server, all that's left is that you just configure it to
export `/mnt/varasto`.

If you're not running a Samba server:

- There's many good tutorials online for setting up Samba
- Or if you're using Docker, you could use my
[joonas-fi/samba](https://github.com/joonas-fi/samba) image that I use to export Varasto
and more.
