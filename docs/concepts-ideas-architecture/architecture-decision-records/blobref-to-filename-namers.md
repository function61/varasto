# BlobRef-to-filename namers

Status: accepted

## Context

Blobstore drivers need a stable filename or object-key component for each `BlobRef`.
The name is part of the physical layout of existing volumes, so changing an algorithm without
a migration makes previously stored blobs unreachable.

The local filesystem driver historically used lowercase extended-hex Base32, split into an
`a/bc/rest` directory shard layout. S3 and Google Drive historically used unpadded URL-safe
Base64 as a flat name. Driver-specific namespaces, such as a local base path, S3 prefix, or
Google Drive parent folder, are not part of the filename algorithm.

## Decision

`blobstore.BlobNamer` supplies the filename component from a `BlobRef`.

Accepted namers:

- `Base32Namer`: lowercase extended-hex Base32 without padding.
- `Base64URLNamer`: URL-safe Base64 without padding.
- `ShardNamer`: composes another namer and applies the `a/bc/rest` shard pattern.

Drivers receive their namer during construction.


## Rejected

Base84 was considered using [its specification](https://00f.net/2026/09/09/base84/) because its
filesystem-safe alphabet can encode a 256-bit reference in 40 to 42 characters, compared with
Base64URL's fixed 43 characters. Its output length varies with the input, which adds unnecessary
layout variability. At 22 million blob references
(about 80 TB of 4 MiB blobs), the expected filename-byte saving is only about 58 MB. Actual
filesystem savings may be smaller because directory entries are allocation-aligned.

The storage reduction does not justify adding a variable-length encoding or another naming format.

## Consequences

The current layouts remain readable without migration. A future mount configuration can select
a `BlobNamer`, but it must retain the selected algorithm for the lifetime of a volume
or provide an explicit migration path.
