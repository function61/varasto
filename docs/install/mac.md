---
title: macOS
---

| Option            | User experience | Occasionally tested by us |
|-------------------|-----------------|---------------------------|
| Docker            | highest         | ☑️                       |
| Linux VM on macOS | medium          | ☐                        |
| Native Varasto server for mac | [lowest](#use-natively) | ☐    |


Use via Docker
--------------

Docker works on macOS so you may have success:

- Installing [Docker for macOS](https://docs.docker.com/docker-for-mac/)
- Continuing with the [Linux (Docker)](linux-docker.md) instructions.
	* Linux because internally `Docker for macOS` uses a Linux VM


Use via Linux VM
----------------

If you already have a Linux VM (or want to set up one for this), then then go back to
[Installation](index.md) and follow the Linux instructions.


Use natively
------------

Varasto server has native compilation to macOS, so this might work. We haven't tested it,
and we don't have service autostart on macOS, so you'll have to start the process manually
each time you start your computer or make an auto-start script yourself.
