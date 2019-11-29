[![Build Status](https://img.shields.io/travis/function61/varasto.svg?style=for-the-badge)](https://travis-ci.org/function61/varasto)
[![Download](https://img.shields.io/badge/Download-bintray%20latest-blue.svg?style=for-the-badge)](https://bintray.com/function61/dl/varasto/_latestVersion#files)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/function61/varasto)

Software defined distributed storage array with custom replication policies and strong
emphasis on integrity and encryption.

See [screenshots](docs/screenshots.md) to get a better picture.

Varasto is optimized for WORM-style (Write Once; Read Many, i.e. file archival, backups)
workloads. If files in your particular directory change more than once an hour averaged
throughout the day, you should have that collection cloned into your computer and have
Varasto take daily (or even hourly) backups. If your files change less often, you can use
Varasto as the authoritative store. 


Status & timeline
-----------------

| Date               | Status      |
|--------------------|-------------|
| Before 30 Nov 2019 | Heavy development, everything subject to change. Not recommended for any data you're willing to lose since data migration scripts won't be made |
| After 30 Nov 2019  | MVP version with rough edges but minimal data loss risk - recommended for early adopters willing to report bugs and feature suggestions to development |
| After 31 Jan 2020  | General availability for all users |

Currently *under heavy development*. Works so robustly (blobs currently cannot be
deleted so if metadata DB is properly backed up, you can't lose data) that I'm already
moving *all my files in*, but I wouldn't yet recommend this for anybody else. Also proper
access controls are nowhere to be found.


Features
--------

| Status | Feature                     | Details                               |
|--------|-----------------------------|---------------------------------------|
| ✓      | Supported OSs               | Linux, Windows (Mac might come later) |
| ✓      | Supported architectures     | amd64, ARM (= PC, Raspberry Pi etc.) |
| ✓      | Backup all your devices' data | Varasto's architecture is ideal for backing up all your PCs, mobile devices etc. |
| ✓      | Supported storage methods   | Local disks or cloud services (AWS S3, Google Drive), all in encrypted form so you don't have to trust the cloud or have data leaks if local HDDs get stolen. |
| ✓      | [Integrated internal database backups](docs/guide_setting-up-backup.md) | Use optional built-in backup to automatically upload encrypted backup of your metadata DB to AWS S3. If you don't like it, there's interface for external backup tools as well. |
| ✓      | Compression                 | Storing well-compressible files? They'll be compressed automatically (if it compresses well) & transparently |
| ✓      | Metadata support & tagging  | Can use metadata sources for automatically fetching movie/TV series info, poster images etc. Can also add tags to collections. |
| ✓      | All files in one place      | Never again forget on which disk a particular file was stored - it's all in one place even if you have 100 disks! Varasto is [dogfooded](https://en.wikipedia.org/wiki/Eating_your_own_dog_food) with ~50 TB of data without any slowdowns. |
| ✓      | Thumbnails for photos       | Automatic thumbnailing of photos/pictures |
| [TODO](https://github.com/function61/varasto/issues/40) | Thumbnails for videos       | Automatic thumbnailing of videos |
| [TODO](https://github.com/function61/varasto/issues/60) | Video & audio transcoding   | Got movie in 4K resolution but your FullHD resolution phone doesn't have the power or bandwidth to watch it? |
| ✓      | Data access methods         | 1) Clone collection to your computer 2) Open/stream files from web UI 3) Access files via network share 4) Access via Linux FUSE interface |
| [TODO](https://github.com/function61/varasto/issues/75) | Atomic snapshots            | Uses LVM on Linux and shadow copies on Windows to grab consistent copies of files |
| ✓      | Data integrity              | Sha256 hashes verified on file write/read - detects bit rot immediately |
| ✓      | Data privacy                | All data is encrypted - each collection with a separate key so compromise of one collection does not compromise other data |
| ✓      | Data sensitivity            | You can mark different collections with different sensitivity levels and decide on login if you want to show only family-friendly content |
| ✓      | Data durability             | Transparently replicates your data to multiple disks / to offsite storage |
| ✓      | Per-collection durability   | To save money, we support storing important files with higher redundancy than less important files |
| ✓      | Transactional               | File or group of files are successfully committed or none at all. Practically no other filesystem does this |
| ✓      | Scheduled scrubbing         | Varasto can scan your disks periodically to detect failing disks ASAP |
| ✓      | [Ransomware protection](docs/guide_ransomware-protection.md) | Run Varasto on a separate security-hardened device/NAS to protect from ransomware, or configure replication to S3 ransomware-protected bucket |
| ✓      | [Integrated SMART monitoring](docs/guide_setting-up-smart-monitoring.md) | Detect disk failures early |
| [TODO](https://github.com/function61/varasto/issues/53) | Tiered storage              | Use SSD for super fast data ingestion, and transfer it in background to a spinning disk |
| [TODO](https://github.com/function61/varasto/issues/39) | Multi-user                  | Have separate file hierarchies for your friends & family |
| TODO   | File sharing                | Share your own files to friends |
| TODO   | Offline drives              | We support use cases where you plug in a particular hard drive occasionally. Queued writes/deletes are applied when volume becomes available |


Docs
----

Design:

- [Terminology](docs/design_terminology.md)
- [Architecture / ideas & goals / inspired by / comparison to similar software](docs/design_architecture-ideas-goals-inspired-by-comparison-to-similar-software.md)

Using:

- [How to install](docs/guide_how-to-install.md)
- [Setting up SMART monitoring](docs/guide_setting-up-smart-monitoring.md)
- [Setting up backup](docs/guide_setting-up-backup.md)
- [Setting up ransomware protection](docs/guide_ransomware-protection.md)
- [(storage) Setting up AWS S3](docs/guide_setting-up-s3.md)
- [(storage) Setting up Google Drive](docs/guide_setting-up-googledrive.md)

Guides for storing different types of content:

- Storing photos (TODO)
- [Storing TV shows](docs/guide_storing-tvshows.md)
- [Storing movies](docs/guide_storing-movies.md)
- Storing podcasts (TODO)

Developers:

- [How to build & develop](https://github.com/function61/turbobob/blob/master/docs/external-how-to-build-and-dev.md)
- [Code documentation on GoDoc.org](https://godoc.org/github.com/function61/varasto)

Misc:

- [Security policy](https://github.com/function61/varasto/security/policy)
- [Sustainability](docs/sustainability.md) (how will this project make money)

Food for thought:

- [How to Remember Your Life by Johnny Harris](https://www.youtube.com/watch?v=GLy4VKeYxD4)
- [How do you organize your data?](https://www.reddit.com/r/DataHoarder/comments/9jz9ln/how_do_you_organize_your_data/)
- [DataHoarder subreddit](https://www.reddit.com/r/DataHoarder/)
- [Timeliner](https://github.com/mholt/timeliner) project archives your Twitter/Facebook
  etc history in structural form with a
  [fantastic motivation](https://github.com/mholt/timeliner#motivation-and-long-term-vision).
