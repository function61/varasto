!!! warning "Managing expectations"
    Varasto is in beta, and thus there are a few missing features and the occasional rough
    edge that you should be aware of. [Read more](#limitations-of-beta-version).


Server installation
-------------------

| Installation                | Fully supported | Notes |
|-----------------------------|-----------------|-------|
| [Linux (Docker)](linux-docker.md) | ☑️ | **Recommended, easiest option**. Only for `x86-64` (you probably have this if you're not sure) |
| [Linux (manual installation)](linux-manual.md) | ☑️ | For Raspberry Pis etc |
| [Windows](windows.md)             | ☐ | |
| Mac                               | ☐ | Docker works on Mac so you may have success using the `Linux (Docker)` guide |


Server vs. client?
------------------

!!! info "Not familiar with the differences of Varasto server and client?"
	[Read about it first](../concepts-ideas-architecture/index.md#client-vs-server)!


Client installation
-------------------

!!! tip "Don't worry about this"
	Once you install the server, the UI will have client download links and help.

This is covered in a [separate document](../using/client/index.md).


Limitations of beta version
---------------------------

Varasto currently has these limitations:

- Access controls (user accounts, authentication) are missing
	* => Do not expose Varasto server to public internet
- Anything mentioned in the
  ["General availability" milestone](https://github.com/function61/varasto/milestone/3)
- Updates between beta versions can be tedious. We'll release instructions, but cannot
  promise that they won't contain manual steps like "download backup, replace this from
  backup file, run this command to import backup".


