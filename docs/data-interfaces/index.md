---
title: Overview on ways to access your data
---

Overview on ways to access your data
====================================

Varasto's web UI is good for quickly:

- fetching the occasional PDF/DOCX/etc. document
- looking at photos
- watching movies/videos
- listening to music.

But there are times when you just need to interact with raw files on the OS level. This is
why we have many interfaces to access your data - each with different pros and cons on use
cases and OS support.


Comparison
----------

| Feature/interface      | [Web UI](web-ui/index.md) | [Network folders](network-folders/index.md) | [Cloning](client/index.md#how-does-the-cloning-interface-look-like) | [FUSE filesystem](fuse/index.md) |
|------------------------|--------|-----------------|---------|-----------------|
| Fast-changing data[^1] | ☐      | ☐              | ☑️     | ☐               |
| Streaming[^2]          | ☑️     | ☑️             | ☐      | ☑️              |
| Open&edit in native apps[^3] | ☐ | On most cases | ☑️      | ☑️             |
| (OS) Linux             | ☑️     | ☑️             | ☑️     | ☑️              |
| (OS) Windows           | ☑️     | ☑️             | ☑️     | ☐               |
| (OS) macOS             | ☑️     | ☑️             | ☑️     | ☐               |
| (OS) Android           | ☑️     | ☑️             | ☐      | ☐               |
| (OS) iOS               | ☑️     | ☑️             | ☐      | ☐               |

!!! tip
	We also have [API for programmatic access](api/index.md), but the above summary is
	focused on end-users.


[^1]: Data that has very frequent changes, cannot be stored in Varasto. The only option is
      to have fast changes happen on another device and have Varasto take periodic snapshots.

[^2]: You don't need to download the entire file collection to your device before using it.
      No storage space is used, because the files are streamed on-demand from Varasto server.

[^3]: How well an interface lets you open&edit a file in your native app - e.g. open a
      photo in Photoshop for editing. Most (but not all apps) work well with network folders -
      but all apps work well with local files (= cloning or FUSE).
