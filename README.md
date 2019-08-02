[![Build Status](https://img.shields.io/travis/function61/varasto.svg?style=for-the-badge)](https://travis-ci.org/function61/varasto)
[![Download](https://img.shields.io/badge/Download-bintray%20latest-blue.svg?style=for-the-badge)](https://bintray.com/function61/dl/varasto/_latestVersion#files)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/function61/varasto)

Software defined distributed storage array with custom replication policies and strong
emphasis on integrity and encryption.

See [screenshots](docs/screenshots.md) to get a better picture.

Status
------

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
| ✓      | Integrated backups          | Use optional built-in backup to automatically upload encrypted backup of your metadata DB to AWS S3 |
| ✓      | Compression                 | Storing well-compressible files? They'll be compressed automatically (if it compressed well) & transparently |
| ✓      | Metadata support            | Can use metadata sources for automatically fetching movie/TV series info, poster images etc. |
| ✓      | All files in one place      | Never again forget on which disk a particular file was stored - it's all in one place even if you have 100 disks! |
| ✓      | Thumbnails for photos       | Automatic thumbnailing of photos/pictures |
| TODO   | Thumbnails for videos       | Automatic thumbnailing of videos |
| TODO   | Video & audio transcoding   | Got movie in 4K resolution but your FullHD resolution phone doesn't have the power or bandwidth to watch it? |
| TODO   | Atomic snapshots            | Uses LVM on Linux and shadow copies on Windows to grab consistent copies of files |
| ✓      | Data integrity              | Sha256 hashes verified on file write/read - detects bit rot immediately |
| ✓      | Data privacy                | All data is encrypted - each collection with a separate key compromise of one collection does not compromise other data |
| ✓      | Data sensitivity            | You can mark different collections with different sensitivity levels and decide on login if you want to show only family-friendly content |
| ✓      | Data durability             | Transparently replicates your data to multiple disks / to offsite storage |
| ✓      | Per-collection durability   | To save money, we support storing important files with higher redundancy than less important files |
| ✓      | Transactional               | File or group of files are successfully committed or none at all. Practically no other filesystem does this |
| ✓      | Scheduled scrubbing         | Varasto can scan your disks periodically to detect failing disks ASAP |
| TODO   | Ransomware protection       | Run Varasto on a separate device to protect from ransomware, or configure replication to S3 versioned bucket |
| ✓      | Where is data stored        | Local disks or cloud storage |
| TODO   | Integrated SMART monitoring | Detect disk failures early |
| TODO   | Tiered storage              | Use SSD for super fast data ingestion, and transfer it in background to a spinning disk |
| TODO   | Multi-user                  | Have separate file hierarchies for your friends & family |
| TODO   | File sharing                | Share your own files to friends |
| TODO   | Offline drives              | We support use cases where you plug in a particular hard drive occasionally. Queued writes/deletes are applied when volume becomes available |


Docs
----

Design:

- [Terminology](docs/design_terminology.md)
- [Architecture / inspired by / comparison to similar software](docs/design_architecture-inspired-by-comparison-to-similar-software.md)

Operating:

- [How to install](docs/guide_how-to-install.md)
- [Setting up Google Drive](docs/guide_setting-up-googledrive.md)
- [Setting up backup](docs/guide_setting-up-backup.md)

Misc:

- [Security policy](https://github.com/function61/varasto/security/policy)


Ideas / goals
-------------

- "[RAID is not backup](https://serverfault.com/questions/2888/why-is-raid-not-a-backup)",
  so you would need backup in addition to RAID anyway. But what if we designed for backup
  first and its redundant storage as the primary source of truth?
- Varasto works like GitHub, with your different directories being like GitHub repos,
  (we call them collections) but with Varasto making automatic commits (= backup interval)
  against them. If you accidentally delete a file, you will find it from a previous
  collection revision. You can "clone" collections you want to work on, to your computer,
  and when you stop working on them you can tell Varasto to delete the local copy and
  Varasto client will ensure that the Varasto server has the latest state before removing.
  This way your end devices can remain almost-stateless. Store locally only the things you
  are working on currently!
- You don't need to clone collections if all you want to do is view files (such as look at
  photo albums, listen to music or watch movies) - Varasto supports streaming too.
- Works on Linux and Windows (mostly due to Go's awesomeness)
- Integrity is the most important thing. Hashes are verified on writing to disk and on
  reading from disk. There is also a scheduled job for checking integrity of your volumes.
- Unified view of all of your data - never again have to remember which disk a particular
  thing was stored on! Got 200 terabytes of data spread across tens of disks? No problem!
- Decoupling metadata from file content. You can move/rename files and folders and modify
  their metadata "offline", i.e. without touching the disk the actual file content is hosted on.
- Configurable encryption. Each collection could have a separate encryption key, which itself
  is asymmetrically encrypted by your personal key which never leaves your hardware security
  module. This way if a hacker MITM's or otherwise learns of a collection-specific
  decryption key, she can't access your other collections. Particularly sensistive collections
  could have such an encryption key even on a file-by-file basis.
- Related to previous point, we should investigate doing as much as possible in the client
  or the browser, so perhaps the decryption keys don't even have to be known by the server.
- Configurable replication policies per collection. Your family photo albums could be
  spread on 2 local disks and 1 AWS S3 bucket, while a movie you ripped from a Blu-ray could
  be only on one disk because in the event of a disk crash, it could be easily recreated.
- Accesses your files by using platform-specific snapshotting
  (LVM on Linux, shadow copies on Windows)
- Kind of like Git or Mercurial but for all of your data, and meant to store all of your
  data in collections (modeled as directories). Version control-like semantics for
  collection history, but "commits" are scheduled instead of explicit. This is meant to
  back up all your data and backups are useless unless they are automated.
- By not operating on (lower) block device level we don't need the complexity of RAID or
  specialized filesystems like ZFS etc. We can use commodity hardware and any operating
  system to reach the desired goals of integrity and availability. If your hard drive ever
  crashes, would you like to try the recovery with striped RAID / parity bits on a
  specialized filesystem, or just a regular NTFS or EXT4?


Philosophy
----------

- [How to Remember Your Life by Johnny Harris](https://www.youtube.com/watch?v=GLy4VKeYxD4)
