Concepts
--------

| Term | Explanation |
|------|-------------|
| Collection | Like a Git repository - stores concrete files and folders. Files consist of multiple blobs. Collections are change tracked by changeset, so if you accidentally deleted or modified a file you can go back in history to restore the file. |
| Directory | Collections are stored in a hierarchy of directories. This is only in metadata sense - directory hierarchy of collections is different than directories inside collections. |
| Blob | Each file is split into 4 MB chunks called blobs. A blob is identified by its content's sha-256. A blob is stored in 1 or more volumes. |
| Volume | A place to store blobs in. A single physical disk, Google Drive, AWS S3 etc. Redundancy is achieved by storing the same blob in separate volumes. |
| Mount | Volume is mounted in a given node, so the node has access to the volume's blobs. Depending on blob driver, same volume can be accessed from multiple nodes. |
| Blob driver | Implements access to different types of volumes. `local-fs` stores blobs in local filesystem. `googledrive` stores in Google Drive etc. |
| Node | One instance of Varasto running on one computer. You can run a cluster of Varasto servers for redundancy and/or convenience (think Varasto running behing firewall at home but also in cloud for remote access). |
| Clone | The act of downloading a collection to your computer for modifying it. Only reading files does not necessarily require cloning as you can stream videos/audio/photos off of Varasto's UI. |
| Push | The act of committing your local changes in a changeset and pushing that changeset to the Varasto server |
| Changeset | A group of changes to multiple files recorded in a single point in time. |
