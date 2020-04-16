We've used Varasto so much that we've witnessed disks die and weird disk failures like a disk
returning corrupt data, claiming no I/O errors - which Varasto refuses to acknowledge.

!!! tip "All disks eventually fail"
	You've to prepare for this. We've compiled instructions to help you resolve any issues.


How does Varasto react to error situations?
-------------------------------------------

| Situation             | Action |
|-----------------------|--------|
| Read corrupted data   | Varasto gives I/O error to user. There is no chance an error is missed even if disk lies and reports OK |
| Write to volume fails | If the volume reports an I/O error, the replica we tried to write won't be recorded in the database |

!!! bug "TODO"
	Varasto should [auto-unmount](https://github.com/function61/varasto/issues/171)
	(maybe configurable) volumes on encountering errors.


Removing a disk from service
----------------------------

When a volume/disk is deemed as about to fail/failed, you should remove it from service.
Do these below steps in order:

### Take a metadata backup

Take a [metadata backup](../metadata-backup/) so that if you make any mistakes, you can
go back. You hopefully have nightly backups, but it's good to take a backup just before
this operation so that if you need to resort to the backup, it contains the most recent changes.

??? question "Why?"
	Marking all volume's data lost involves "forgetting" which blobs were stored on the volume
	you're about to decommission - i.e. Varasto will think the volume is empty. This is so
	that replication controller will notice the discrepancy between desired and actual state
	and start demanding new replicas to make your data safe again. This will be
	[further explained here](#reconcile-any-lost-redundancy).


### Stop writing new data to the failing disk

Remove the volume from any replication policies' `New data goes to` plan.

??? info "UI screenshot"
	![](../replication-policies/screenshot.png)


### Mark all volume data lost

TODO: have the `mark all volume data lost` command assert that replication policies won't
refer to it.

!!! tip
	For your safety, the command has a `Only if redundancy` switch so Varasto aborts the
	command if it would lose the last redundant replica of any file. This is so that you
	have one last chance to decide to try to recover data from a disk if it comes as a
	surprise to you that your replication policies didn't spread the data redundantly enough.


### Reconcile any lost redundancy

Go to `Settings` > `Replication policies` > `Reconciliation` to start a scan that will
notice which data will conflict with the desired replica count and require you to specify
which volume will be used to fulfill the storage needs of the conflicted replica count.

!!! tip
	This scan is done automatically once a day, so it's not a disaster if you forget to do it.
	But the risk is lower if you scan immediately instead of waiting for the alert to be raised.


### Decommission the volume

TODO: rename to "hide a volume"?

This action only hides a volume from UI in places where you usually only want to see
in-use volumes.

You may now detach the disk and get rid of it. Don't worry about secure disposal of the
disk, since all the data was encrypted anyway. (provided you used the disk only for
Varasto data)
