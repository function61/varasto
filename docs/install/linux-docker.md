Linux (Docker)
==============

!!! info "Prerequisite: Docker"
	If you're on Ubuntu, it's one `$ sudo apt-get install docker.io` away. For other distros,
	[see instructions](https://docs.docker.com/engine/install/).

We have two Docker start examples available:

- Direct `$ docker run ...` command
- Docker compose file
	* If you're not familiar with this, you might want to start with the run command. You
	  can switch to using the compose file later if you want.


Start Varasto
-------------

=== "Docker run"
	Run:

	```console
	docker run -d --name varasto \
		-v varasto-db:/varasto-db \
		-v /mnt/varasto:/mnt/varasto:shared \
		-v /dev:/dev:ro \
		--privileged \
		--device /dev/fuse \
		-p 443:443 \
		fn61/varasto
	```

	??? info "Explanations for the Docker options"
		The options are documented in the Docker compose file (the other tab)

=== "Docker compose"
	Save as `docker-compose.yml`:
	
	```yaml
	version: "3.5"
	services:
	  varasto:
	    image: fn61/varasto
	    cap_add:
	    - SYS_ADMIN              # for FUSE support. these are not required if you have "privileged: true"
	    - MKNOD                  # (but are IF you remove privileged because you don't need SMART support)
	    devices:
	    - /dev/fuse              # for FUSE support
	    privileged: true         # for SMART support
	    ports:
	    - "443:443"              # Varasto network port (https). Change first number if you have this reserved
	    volumes:
	    - type: volume
	      source: varasto-db     # Varasto state. Useful to be a named volume so version updates are easier.
	      target: /varasto-db
	    - type: bind
	      source: /mnt/varasto
	      target: /mnt/varasto
	      bind:
	        propagation: shared  # For sub-mounts (FUSE) to be visible to the host
	    - type: bind
	      source: /dev           # SMART support requires access to raw disks
	      target: /dev
	      read_only: true
	volumes:
	  varasto-db:
	    # so docker-compose won't try to generate the concrete volume based on this .yml file's
	    # directory. (not much else reason than to prevent directory rename from starting
	    # with a new database when you're updating Varasto)
	    external: true
	```
	
	Then start Varasto:
	
	```console
	docker-compose up -d
	```


After Varasto is started
------------------------

Now you can navigate your browser to `https://localhost/` and **hit "Help" from the menu
to reach the getting started wizard** which will help you set up everything.

(You'll have to approve the "insecure certificate" warning.)


Version pinning
---------------

We offer both:

- The `latest` tag (always points to latest stable release)
	* Docker defaults to this when just running image without a specific tag - which is
	  what our examples in this page did.
- Version-specific image tags

If you want to be explicit about the version (version pinning), replace `fn61/varasto` with
`fn61/varasto:<version>`. The version names are the same as
[our release names](https://github.com/function61/varasto/releases).

!!! warning
	If you visit [our Docker Hub page](https://hub.docker.com/r/fn61/varasto) it will also
	have development builds there. To be safe, only use the stable releases from the
	releases page!


Troubleshooting
---------------

### Varasto doesn't start or I can't reach its Web UI

Check the logs to see what's the problem:

```console
docker logs varasto
```
