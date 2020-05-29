Limitations of beta version
===========================

Varasto currently has these limitations:

- Access controls (user accounts, authentication) are missing
	* => Do not expose Varasto server to public internet
- Anything mentioned in the
  ["General availability" milestone](https://github.com/function61/varasto/milestone/3)
- While deletes are supported, file history deletion is not yet supported, so you can't
  reclaim any space by deleting files yet.
- Updates between beta versions can be tedious. We'll release instructions, but cannot
  promise that they won't contain manual steps like "download backup, replace this from
  backup file, run this command to import backup".
