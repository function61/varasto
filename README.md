![Build status](https://github.com/function61/varasto/workflows/Build/badge.svg)
[![Download & install](docs/assets/Download-install-green.svg)](https://function61.com/varasto/docs/install/)

All your files in one simple, replicated, encrypted place - with built-in backups and
configurable ransomware protection.

Vision
------

![Varasto vision](docs/vision.png)


Video & screenshot introduction
-------------------------------

TODO: video

See [screenshots](https://function61.com/varasto/docs/screenshots/) to get a better picture.


Website & documentation
-----------------------

We have wonderful documentation on [our website](https://function61.com/varasto/).


Features
--------

**NOTE**: Our [documentation](https://function61.com/varasto/docs/#features) has more
details & links in this table.

|                             | Details                               |
|-----------------------------|---------------------------------------|
| Supported OSes              | Almost everything: PCs, mobile devices (Android, iOS), Raspberry Pis etc. |
| Data privacy                | All data is encrypted - each collection with a separate key so compromise of one collection does not compromise other data. Take back ownership of your data. |
| Data durability             | Transparently replicates your data to multiple disks / off-site storage. |
| Data integrity              | `SHA-256` hashes verified on file write/read - detects [bit rot](https://en.wikipedia.org/wiki/Data_degradation) immediately. We also have scheduled scrubbing to detect errors in the background before they affect you. |
| Data sensitivity            | You can mark different collections with different sensitivity levels and decide on login which sensitivity level content do you want to show. |
| Backup all your devices' data | Varasto's architecture is ideal for backing up all your PCs, mobile devices etc. |
| Supported storage methods   | Local disks or cloud services (AWS S3, Google Drive), all in encrypted form so you don't have to trust the cloud ("zero trust" model) or have data leaks if local disks get stolen. |
| Data access methods         | 1) Clone collection to your computer 2) Open/stream files from web UI 3) Via network folders 4) Linux FUSE interface |
| Integrated metadata backups | Use optional built-in backup to automatically upload encrypted backup of your metadata DB to AWS S3. If you don't like it, there's interface for external backup tools as well. |
| Transparent compression     | Only well-compressible files will be automatically compressed |
| Metadata support & tagging  | Can use metadata sources for automatically fetching movie/TV series info, poster images etc. Can also add tags to collections. |
| All files in one place      | Never again forget on which disk a particular file was stored - it's all in one place even if you have 100 disks! Varasto is [dogfooded](https://en.wikipedia.org/wiki/Eating_your_own_dog_food) with ~50 TB of data without any slowdowns. |
| Thumbnails for photos       | Automatic thumbnailing of photos/pictures |
| Health monitoring           | Get warnings or alerts if there is anything wrong with your volumes, data or Varasto. |
| Per-collection durability   | To save money, we support storing important files with higher redundancy than less important files |
| Transactional               | File or group of files are successfully committed or none at all. Practically no other filesystem does this |
| Ransomware protection       | Run Varasto on a separate security-hardened device/NAS to protect from ransomware, or configure replication to S3 ransomware-protected bucket |
| Integrated SMART monitoring | Detect disk failures early |
