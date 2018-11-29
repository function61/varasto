Software defined distributed storage array with custom replication policies and strong
emphasis on integrity and encryption.


NOTE: Name "Bup" is temporary working name, and *will change*.
[Bup](https://github.com/bup/bup) is already taken by a very similar project.


Ideas / goals
-------------

- Works on Linux and Windows (mostly due to Go's awesomeness)
- Integrity is the most important thing. Hashes are verified on writing to disk and on
  reading from disk.
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


Inspired by & alternative software
----------------------------------

- [Syncthing](https://syncthing.net/)
- [Duplicati](https://www.duplicati.com/)
- [restic](https://restic.net/)
- [bup](https://github.com/bup/bup)
- [Perkeep](https://perkeep.org/doc/overview)
- [upspin](https://upspin.io/doc/arch.md)

