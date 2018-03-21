## GoBonnieGo

GoBonnieGo is a _minimal_ Golang implementation of Tim Bray's
[bonnie](https://code.google.com/p/bonnie-64/) (*bonnie* is
written in C).

It measures disk throughput by reading and writing files.

It presents three disk metrics:

1. Sequential Write (higher is better)
2. Sequential Read (higher is better)
3. IOPS (I/O Operations per Second) (higher is better)

## Getting GoBonnieGo

The easiest way to get GoBonnieGo is to download the pre-built binaries in the
[Releases](https://github.com/cunnie/gobonniego/releases/) section.  In the
following example, we are logged into a Linux box and we download and run the
Linux binary:

```
curl -o gobonniego -L https://github.com/cunnie/gobonniego/releases/download/1.0.7/gobonniego-linux-amd64
chmod +x gobonniego
./gobonniego
```

Alternatively, you can run GoBonnieGo from source if you're a Golang developer:

```
go get github.com/cunnie/gobonniego
cd $GOPATH/src/github.com/cunnie/gobonniego
go run gobonniego/gobonniego.go  # "Go Bonnie Go, Go"!
```

## Examples

GoBonnieGo can be invoked without parameters; its defaults are reasonable.

```
gobonniego
```

Typical output:

```
2018/02/19 12:03:16 gobonniego starting. version: 1.0.7, threads: 8, disk space to use (MiB): 3984
Sequential Write MB/s: 748.22
Sequential Read MB/s: 1025.19
IOPS: 23832
```

Running with the verbose option (`-v`) will print additional timestamped information
to STDERR:

```
gobonniego -v
```

Yields:

```
2018/02/24 17:20:20 gobonniego starting. version: 1.0.7, threads: 8, disk space to use (MiB): 512
2018/02/24 17:20:20 Number of CPU cores: 8
2018/02/24 17:20:20 Total system RAM (MiB): 65536
2018/02/24 17:20:20 Bonnie working directory: /var/folders/lp/k0g2hcfs0bz1c4zn90pnh32w0000gn/T/gobonniegoParent337382325
2018/02/24 17:20:21 Written (MiB): 512
2018/02/24 17:20:21 Written (MB): 536.870912
2018/02/24 17:20:21 Duration (seconds): 1.029243
Sequential Write MB/s: 521.62
2018/02/24 17:20:21 Read (MiB): 512
2018/02/24 17:20:21 Read (MB): 536.870912
2018/02/24 17:20:21 Duration (seconds): 0.023219
Sequential Read MB/s: 23121.95
2018/02/24 17:20:37 operations 16927770
2018/02/24 17:20:37 Duration (seconds): 15.940455
IOPS: 1061938
```

You may specify the number of test runs. This is useful when gathering a large
sample set.

```
gobonniego -v -runs 2
```

You may specify the placement of GoBonnieGo's test files. This is useful if
the default filesystem is too small or if you want to test a specific
filesystem/disk.  GoBonnieGo will clean up after itself, and will not delete
the directory it's told to run in (you can safely specify `/tmp` or `/` as the
directory). Here are some examples:

```
gobonniego -dir D:\
gobonniego -dir /tmp
gobonniego -dir /zfs/tank
gobonniego -dir /Volumes/USB
gobonniego -dir /var/vcap/store/
```

You may specify the number of threads (Goroutines) to run with the `-threads`
flag. In this example, we spawn 8 threads:

```
gobonniego -threads 8
```

You may choose to have JSON-formatted output by specifying the `-json` flag. In
the following example, we pass the JSON output to the popular
[`jq`](https://stedolan.github.io/jq/) tool which prettifies the JSON output:

```
gobonniego -json | jq -r .
```

Yields:

```json
{
  "version": "1.0.7",
  "gobonniego_directory": "/var/folders/zp/vmj1nyzj6p567k5syt3hvq3h0000gn/T/gobonniegoParent983654097/gobonniego",
  "disk_space_used_gib": 0.5,
  "num_readers_and_writers": 4,
  "physical_memory_bytes": 17179869184,
  "iops_duration_seconds": 0.5,
  "results": [
    {
      "write_megabytes_per_second": 1608.7642096164034,
      "read_megabytes_per_second": 11717.751585397944,
      "iops": 307918.5948218495,
      "write_bytes": 536870912,
      "write_nanoseconds": 333716345,
      "read_bytes": 536870912,
      "read_nanoseconds": 45816888,
      "io_operations": 222180,
      "io_nanoseconds": 721554345
    }
  ]
}
```


You may specify the amount of disk space GoBonnieGo should use with the `-size` flag
which takes an integer argument (in GiB). This can be used to iterate rapidly while testing.
For example, to constrain GoBonnieGo to use  0.5 GiB of disk space, type the following:

```
gobonniego -size 0.5
```

You may specify the duration of the IOPS test. By default it runs for 5
seconds, but this can be overridden in order to iterate rapidly while testing.
For example, to trim the duration of the IOPS test to 1/2 second, type the
following:

```
gobonniego --iops-duration=0.5
```

`-version` will display the current version of GoBonnieGo:

```
gobonniego -version
```

Yields:

```
gobonniego version 1.0.7
```

`gobonniego -h` will print out the available command line options and their
current default values:

```
Usage of ./gobonniego:
  -dir string
        The directory in which gobonniego places its temporary files, should have at least '-size' space available (default "/var/folders/zp/vmj1nyzj6p567k5syt3hvq3h0000gn/T/gobonniegoParent120217156")
  -iops-duration float
        The duration in seconds to run the IOPS benchmark, set to 0.5 for quick feedback during development (default 15)
  -json
        Version. Will print JSON-formatted results to stdout. Does not affect diagnostics to stderr
  -runs int
        The number of test runs (default 1)
  -size float
        The amount of disk space to use (in GiB), defaults to twice the physical RAM (default 32)
  -threads int
        The number of concurrent readers/writers, defaults to the number of CPU cores (default 4)
  -v    Verbose. Will print to stderr diagnostic information such as the amount of RAM, number of cores, etc.
  -version
        Version. Will print the current version of gobonniego and then exit
```

## Technical Notes

GoBonnieGo detects the number of CPU cores and the amount of RAM.

The number of cores may not match the number of physical cores. For example, an
Intel core i5 with two physical cores and hyperthreading is detected as 4
cores.

GoBonnieGo spawns one thread for each core unless overridden by the `-threads`
flag.

GoBonnieGo writes twice the amount of RAM unless overridden with the `-size`
flag.  For example, on a system with 16 GiB of RAM, GoBonnieGo would write 32
GiB of data. This is to reduce the effect of the [buffer
cache](http://www.tldp.org/LDP/sag/html/buffer-cache.html), which may give
misleadingly good results.

If the sequential read performance is several multiples of the sequential write
performance, it's likely that the buffer cache has skewed the results.

The buffer cache also skews the results of the IOPS metric — the number reported
by GoBonnieGo is often much too high, and a reasonable rule of thumb would be to
**halve the IOPS value reported by GoBonnieGo** (e.g. 200k IOPS would become
100k IOPS) (assumptions: GoBonnieGo dataset size is twice RAM, that half the
dataset is in the buffer cache, that any given operation has a 50% chance of
hitting the cache instead of the disk, that the operation is a read (true 90% of
the time), and that any operation hitting the buffer cache returns
instantaneously (takes zero seconds to process)).

If run as root on Linux or macOS systems, GoBonnieGo will flush the buffer cache
before running the read test or the IOPS test. It accomplishes this on linux by
writing `3` to [`/proc/sys/vm/drop_caches`](https://linux-mm.org/Drop_Caches);
on macOS, it runs the
[`purge`](https://developer.apple.com/legacy/library/documentation/Darwin/Reference/ManPages/man8/purge.8.html)
command. The results given by GoBonnieGo under these conditions will more
closely reflect the performance of the underlying hardware (i.e. you should not
halve the IOPS value), but there is always a risk when running commands as root.
_Caveat Utor_.

GoBonnieGo divides the total amount to write by the number of threads. For
example, a 4-core system with 8 GiB of RAM would have four threads each of
which would concurrently write 4 GiB of data for a total of 16 GiB.

GoBonnieGo writes with buffered I/O; however, it waits for
[`bufio.Flush()`](https://golang.org/pkg/bufio/#Writer.Flush) to complete
before recording the duration.

GoBonnieGo creates a 64 kiB chunk of random data which it writes in
succession to disk.  It's random in order to avoid inflating the results for
filesystems which enable compression (e.g. ZFS). We are aware that we are
unfairly handicapping filesystems which enable compression.

GoBonnieGo reads the files concurrently in 64 kiB chunks. Every 127 chunks it
does a byte comparison against the original random data 64 kiB chunk to make
sure there has been no corruption. This probably exacts a small penalty in read
performance.

For IOPS measurement, a GoBonnieGo thread seeks to a random position in the
file and reads 512 bytes. This counts as a single operation. Every tenth seek
instead of reading it will write 512 bytes of data. This also counts as an
operation. The ratio of reads:writes is 10:1, in order to approximate the ratio
that the TPC-E benchmark uses
(<http://www.cs.cmu.edu/~chensm/papers/TPCE-sigmod-record10.pdf>).

The IOPS measurement cycle runs for approximately 15 seconds, at the end of
which GoBonnieGo tallies up the number of I/O operations and divides by the
duration of the test.

GoBonnieGo uses
[`ioutil.TempDir()`](https://golang.org/pkg/io/ioutil/#TempDir) to create the
temporary directory in which to place its files, unless overridden by the
`-dir` flag. On Linux systems this temporary directory is often `/tmp/`, on
macOS systems, `/var/folders/...`.

GoBonnieGo measures bytes in [MiB](https://en.wikipedia.org/wiki/Mebibyte)
and GiB:

- 1 MiB == 2<sup>20</sup> bytes == 1,048,576 bytes
- 1 GiB == 2<sup>30</sup> bytes == 1,073,741,824 bytes

However, the output of the read and write metrics are in MB/s
(Megabytes/second, i.e. 1,000,000 bytes per second) to conform with the
industry norm.

## Bugs

If GoBonnieGo crashes you may need to find and delete the GoBonnieGo files
manually. Below is a sample `find` command to locate the GoBonnieGo
directory; delete that directory and everything underneath:

```
find / -name gobonniegoParent\* -follow
```

GoBonnieGo needs integration tests. Badly.

### Acknowledgements

[Tim Bray](https://www.tbray.org/ongoing/) wrote the original `bonnie` which
inspired Russell Coker to write
[`bonnie++`](https://www.coker.com.au/bonnie++/) which was used to measure ZFS
performance in calomel.org's
[post](https://calomel.org/zfs_raid_speed_capacity.html) which inspired me to
build a ZFS-based
[NAS](https://content.pivotal.io/blog/a-high-performing-mid-range-nas-server)
and
[benchmark](https://content.pivotal.io/blog/a-high-performing-mid-range-nas-server-part-2-performance-tuning-for-iscsi)
it. And credit must be given to Brendan Gregg's excellent post, _[Active
Benchmarking:
Bonnie++](http://www.brendangregg.com/ActiveBenchmarking/bonnie++.html)_.


### Name

Tim Bray suggested the name, _gobonniego_:

> maybe "GoBonnieGo"?

It's a reference to the refrain of Chuck Berry's song, [Johnny B.
Goode](https://en.wikipedia.org/wiki/Johnny_B._Goode), which repeats the
phrase, "Go Johnny go"

### Impetus

The impetus for writing GoBonnieGo is to provide concurrency.  During a
benchmark of a ZFS filesystem (using *bonnie++*), it became clear that a the
single-threaded performance of *bonnie++* and not disk speed was the limiting
factor.
