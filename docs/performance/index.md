Benchmarks
----------

| TLS [^1] | Test             | Write [MiB/s] | Read [MiB/s] |
|--------|------------------|---------------|--------------|
| ☐ | One large file [^2]   | 299.2         | 220          |
| ☐ | Many small files | TODO          | TODO         |
| ☑️ | One large file   | 209.0         | 162          |

Measured inside a VirtualBox Ubuntu Linux VM (host: Windows 10). VM had access to 6 threads,
CPU ([AMD Ryzen 5 2400G](https://www.cpubenchmark.net/cpu.php?cpu=AMD+Ryzen+5+2400G&id=3183),
not a top-spec CPU) had 8 threads.


Notes on benchmarks
-------------------

### TODO: include client benchmarks

Currently the benchmark focuses on the server component alone, i.e. the Varasto client is
not included in the benchmark (we're using `$ curl` as the client so TCP/HTTP overhead is
accounted though).


### Future performance focus

Varasto hasn't had much performance tuning done, so there may be major improvements down
the line.


### Architecture improvements may improve performance

Varasto has client-server architecture. Currently the server does most of the expensive
operations like:

- calculating `sha-256` hashes
- encryption

[These will be moved to the client](https://github.com/function61/varasto/issues/133),
and that will make the server's job considerably easier (**= increase server's read/write
speed**).

If you have:

- Just a single client to a single server, this won't make much of a difference
  overall (assuming they have similar CPU performance OR are the same machine).

- Many clients, then this future change will increase the overall throughput since most of
  the heavy work will be outsourced to the clients.


Testing methodology
-------------------

### Write testing

- Uploader (`$ curl`) reads a cached file from RAM
	* To not measure my particular disk speed/latency

- Varasto writes to a RAMdisk via Varasto's `LocalFS` blob driver
	* To not measure my particular disk speed/latency

- The API has a special endpoint for file uploads from browser (`/fileupload`). That
  endpoint doesn't use parallelization in Varasto's master branch - so for benchmarking I
  patched it to support parallelization because that's more closely in line with the
  standard interface (`POST /blobs/{ref}`) which supports parallelization.
	* This discrepancy will get addressed when we start benchmarking from the actual client.

Command:

```console
$ fname="VID_20190310_140702.mp4" && curl -X POST --data-binary @$fname "http://localhost/api_v2/collections/EQi_3OhROUs/fileupload?mtime=1522337989000&filename=$fname"
```

### Read testing

We'll read the same large file from Varasto that we previously wrote. Varasto reads it
from RAMdisk (but still does the usual decryption + integrity verification).

Command:

```console
$ curl http://localhost/api_v2/collections/EQi_3OhROUs/head/dl?file=VID_20190310_140702.mp4 | pv > /dev/null
```

[^1]: Whether the transport between the client and the server was https
[^2]: 595 MiB file
