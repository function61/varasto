Setting up backup
=================

Motivation
----------

You don't want to lose any of your files/data.

Everything Varasto stores is encrypted on-disk, so if you lose the encryption keys you'll
lose the data forever. The encryption keys are stored in the metadata database in Varasto,
so backing it up is essential.

> If you don't back up your metadata database, you are OK with losing your files.


Setting up
----------

Since this is so important, we've made it easy to back it up - Varasto has
[µbackup](https://github.com/function61/ubackup) built-in. The backup files are encrypted
so your backup hosting provider can't look at your metadata.

µbackup writes a copy of your metadata database in AWS S3. You'll want to back up in a remote
location to protect from fires or power surge events that could destroy your hardware.

NOTE: this process is for advanced users. Maybe in the future we'll have an easier UI for this.

Look at µbackup docs for the decryption key generation, and use the `print-default-config`
verb to get yourself a config template. You'll need to put the `"config": ...` section
in Varasto's config so it looks like this:

```
{
	"db_location": "varasto.db",
	...
	"backup_config": {
	    "bucket": "myname-backups",
	    "bucket_region": "us-east-1",
	    "access_key_id": "AKIA.....",
	    "access_key_secret": "........................................",
	    "encryption_publickey": "-----BEGIN RSA PUBLIC KEY-----\n....EAAQ==\n-----END RSA PUBLIC KEY-----\n"
	}
}

```


Taking a backup
---------------

In Varasto's UI (on "Server info" page) there's a "Backup database" button. It'll start
the backup, but currently you'll only see its progress from the logs.

It's fine to use Varasto during backing up, since the underlying database uses
[MVCC](https://en.wikipedia.org/wiki/Multiversion_concurrency_control).


Restoring from backup, motivation for testing
---------------------------------------------

> "Nobody wants backup, they only want restore."

i.e. backups are useless unless you can use them to restore data. That's why you should
periodically test restoring your backups to verify that they work.

> "Schrodinger Backups: The condition of any backup is unknown until a restore is attempted"


Restoring from backup, instructions
-----------------------------------

µbackup has instructions on how to obtain the backup file. Make sure you decrypt the backup
file before trying to import the backup file.

To restore a database, run:

```
$ sto server dbimport < backupfile
```


Using external backup program
-----------------------------

If you don't want to use the built-in µbackup, there's an interface for other backup programs.

You can get a consistent metadata DB backup from the following REST endpoint:

```
$ curl -H 'Authorization: Bearer ...' http://localhost:8066/api_v2/database/export > export.log
```

You can get a bearer token by visiting `Settings > Clients` and creating a new client.
`Backup program` would be a descriptive name.
