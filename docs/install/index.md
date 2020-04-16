!!! warning "Managing expectations"
    Varasto is in beta, and thus there are a few missing features and the occasional rough
    edge that you should be aware of. [Read more](limitations-of-beta-version.md).


Server installation
-------------------

| Installation                | Fully supported[^1] | Notes |
|-----------------------------|-----------------|-------|
| [Linux (Docker)](linux-docker.md) | ☑️ | **Recommended, easiest option**. Only for `x86-64` (you probably have this if you're not sure) |
| [Linux (manual installation)](linux-manual.md) | ☑️ | For users not wanting to use Docker **OR** using Raspberry Pis etc (our Docker image will support non-`x86-64` arches soon) |
| [Windows](windows.md)             | ☐ | |
| [macOS](mac.md)                     | ☐ | |


The client?
-----------

!!! info "Not familiar with the differences of Varasto server and client?"
	[Read about it first](../concepts-ideas-architecture/index.md#client-vs-server)!

!!! tip "Don't worry about this"
	Once you install the server, the UI will have client download links and help.
	
	If you want to dig in anyway, there's a [separate document](../data-interfaces/client/index.md).


[^1]: Extensively tested by us and most likely to work as intended.
