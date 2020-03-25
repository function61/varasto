Replication policies
====================

A replication policy specifies on which volumes a collection's files should be stored on.

![Drawing](replication-policies.png)


Do I need multiple policies?
----------------------------

If you have data with varying types of importance, you can have for example:

- A default replication policy that saves data to three disks
  * The default policy is specified for root directory and is used for all file collections
    unless a directory subtree explicitly specifies a different policy
- A replication policy for one subdirectory three (e.g. "All movies") for less important
  data that will be saved onto just one disk
