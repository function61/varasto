---
title: Update
---

You probably reached this page because Varasto told you there's a new version available.

It's time to update to the latest version of Varasto.


Changelog
---------

If you want to find out about the latest changes, check out our
[releases page](https://github.com/function61/varasto/releases).


Before updating
---------------

### Take a backup

Before updating, [take a backup](../../using/metadata-backup/index.md#taking-a-backup-manually)
(and verify that it was successfull).


How to update
-------------

This one depends on how you installed Varasto:

=== "Docker"
	Updating means restarting Varasto with an updated version of its Docker image.

	Stop Varasto:

	```console
	docker stop varasto
	```

	Remove the container (so we can run Varasto again with a different image version):

	```console
	docker rm varasto
	```

	!!! tip
		Don't worry, removing the Varasto container doesn't remove Varasto content / state
		(= metadata database), since we gave it a named volume when we started it.

	Pull newest version of Varasto image:

	```console
	docker pull fn61/varasto
	```

	Now start the new version of Varasto using the same command as in the
	[original installation instructions](../linux-docker.md).

=== "Docker compose"
	Updating means restarting Varasto with an updated version of its Docker image.

	Go in the directory where you stored the Varasto's `docker-compose.yml` (per the
	install instructions):

	```console
	cd somewhere/
	```

	Stop Varasto:

	```console
	docker-compose down
	```

	Pull newest version of Varasto image:

	```console
	docker pull fn61/varasto
	```

	Now start the new version of Varasto using the same command as in the
	[original installation instructions](../linux-docker.md).

=== "Manual installation"
	Updating means replacing the `sto` binary and `public.tar.gz` with newer versions.

	!!! info "This guide assumes Linux"
		If you're on some other system, improvise.

	Stop Varasto. If you're using systemd:

	```console
	systemctl stop varasto
	```

	Delete the old executable, `public.tar.gz` archive and its extracted directory:

	```console
	rm -rf sto public.tar.gz public/
	```

	You can go over the [original installation instructions](../linux-manual.md) if you want,
	but only do these:

	- download the binary
	- and optionally the `public.tar.gz` file (it's downloaded automatically if it's not found)

	(i.e. configuration and service auto-start installation should not be done again)

	Start Varasto.
