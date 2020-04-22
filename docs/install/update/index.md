Update
======

You probably reached this page because Varasto told you there's a new version available.

It's time to update to the latest version of Varasto.


Before updating
---------------

### Take a backup

Before updating, [take a backup](../../using/metadata-backup/index.md#taking-a-backup-manually)
(and verify that it was successfull).


Changelog
---------

To find out about the latest release, along with its changes, check out our
[releases page](https://github.com/function61/varasto/releases).


How to update
-------------

This one depends on how you installed Varasto.


### Docker

Updating means restarting Varasto with an updated version of its Docker image.

!!! info "Docker and 'latest' image"
	If you launched Varasto without specifying which version of the image to use (if you're
	unsure, then you didn't), then Docker defaults to `latest` version.
	
	The `latest` Docker tag is "dynamic" - i.e. the actual image it points to changes
	over time. In this case run `$ docker pull fn61/varasto` to get the latest "latest"
	version!

If you explicitly specified version of Varasto that you want to use, then you don't have
to run the pull command.

Stop Varasto:

```console
docker stop varasto
```

Remove the container (so we can run Varasto again with a different image):

```console
docker rm varasto
```

Now start new version of Varasto. Use the same command as in the
[original installation instructions](../linux-docker.md), and:

- If you used the `latest` version, you can run the exact same command
- If you explicitly defined version, then find out the newest version from our
  [releases page](#changelog) (Docker tag names are same as our release names).



### Manual installation

!!! tip "Summary"
	Updating boils down to replacing the `sto` binary and `public.tar.gz` with newer versions.

Stop Varasto.

Delete the old:

- `sto` binary file
- `public.tar.gz` archive
- `public/` directory (this is the above archive extracted)

You can go over the [original installation instructions](../linux-manual.md) if you want,
but only download the binary and the `public.tar.gz` files (configuration and service
auto-start installation should not be done again).

Start Varasto.
