This directory contains the persisted data in the first generation format, that we'll
forever support migrating from, to newer formats.

These files are intended for Varasto development, testing and ensuring that our migrations
work from the very first generation of data.

The files are:

```
.
|-- blob-volumes.tar
|-- varastoclient-config.json
`-- varasto.db
```

Client config (`varastoclient-config.json`) is needed for some server subsystems as well.

The tar contains two volumes. It has all the collection's blobs in volume A, and volume B
is empty (it has the volume UUID though, so it can be mounted):

```
.
|-- vol-a
|   |-- 0
|   |   `-- 00
|   |       `-- 0000000000000000000000000000000000000000000000000
|   |-- 8
|   |   `-- 0s
|   |       `-- 30qfrbdvet2aq8qdmsr1tkvqgf68d4edc5a83r2sm0293lsng
|   |-- c
|   |   `-- sq
|   |       `-- j1hn93jkm496k2b019fkl625h47e4lgr2sbukdg0l42ss65qg
|   |-- d
|   |   `-- d8
|   |       `-- 5vtg38dc3qp4v55qknbluao4sfi4adtuq44pc63aqfi3er5j0
|   `-- k
|       `-- r8
|           `-- m8sd6e7e90c8k70lr0ib84rsasc1l69k8ad83ajv38vbf1thg
`-- vol-b
    `-- 0
        `-- 00
            `-- 0000000000000000000000000000000000000000000000000
```
