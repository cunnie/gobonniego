## Go Bonnie Go!

`gobonniego` is a _minimal_ Golang implementation of Tim Bray's
[bonnie](https://code.google.com/p/bonnie-64/) (*bonnie* is
written in C).

It measures disk throughput by reading and writing files.

It presents three disk metrics:

1. Sequential Write (higher is better)
2. Sequential Read (higher is better)
3. IOPS (I/O Operations per Second) (higher is better)

## Getting `gobonniego`

The easiest way to get `gobonniego` is to download the pre-built binaries in the
[Releases](https://github.com/cunnie/gobonniego/releases/) section.  In the
following example, we are logged into a Linux box and we download and run the
Linux binary:

```
curl -o gobonniego -L https://github.com/cunnie/gobonniego/releases/download/1.0.2/gobonniego-linux-amd64
chmod +x gobonniego
./gobonniego
```

Alternatively, you can run `gobonniego` from source if you're a Golang developer:

```
go get github.com/cunnie/gobonniego
cd $GOPATH/src/github.com/cunnie/gobonniego
go run gobonniego.go  # "Go Bonnie Go, Go"!
```

## Examples

`gobonniego` can be invoked without parameters; its defaults are reasonable.

```
gobonniego
```

Typical output:

```
Sequential Write MiB/s: 1229.10
Sequential Read MiB/s: 6729.41
IOPS: 26156
```

Running with the verbose option (`-v`) will print additional timestamped information
to STDERR:

```
gobonniego -v
```

Yields:

```
2018/02/17 09:48:35 Number of CPU cores: 8
2018/02/17 09:48:35 Number of concurrent threads: 8
2018/02/17 09:48:35 Total system RAM (MiB): 1024
2018/02/17 09:48:35 Amount of disk space to be used during test (MiB): 1024
2018/02/17 09:48:35 Bonnie working directory: /var/folders/lp/k0g2hcfs0bz1c4zn90pnh32w0000gn/T/gobonniegoParent275808164/gobonniego
2018/02/17 09:48:38 Written (MiB): 1024
2018/02/17 09:48:38 Duration (seconds): 2.556555
Sequential Write MiB/s: 400.54
2018/02/17 09:48:38 Read (MiB): 1024
2018/02/17 09:48:38 Duration (seconds): 0.063138
Sequential Read MiB/s: 16218.43
2018/02/17 09:48:53 operations 524288
2018/02/17 09:48:53 Duration (seconds): 15.017886
IOPS: 34911
```

You can specify the placement of `gobonniego`'s test files. This is useful if the
default filesystem is too small or if you want to test a specific disk.
`gobonniego` will clean up after itself, and will not delete the directory it's
told to run in (you can safely specify `/tmp` or `/` as the directory). Here
are some examples:

```
gobonniego -dir D:\
gobonniego -dir /tmp
gobonniego -dir /zfs/tank
gobonniego -dir /Volumes/USB
```

You may specify the number of threads (Goroutines) to run with the `-threads`
flag. In this example, we spawn 8 threads:

```
gobonniego -threads 8
```

`-version` will display the current version of `gobonniego`:

```
gobonniego -version
```

Yields:

```
gobonniego version 1.0.2
```

`gobonniego -h` will print out the available command line options and their
current default values:

```
Usage of /var/folders/lp/k0g2hcfs0bz1c4zn90pnh32w0000gn/T/go-build844154903/b001/exe/gobonniego:
  -dir string
    	The directory in which gobonniego places its temp files, should have at least twice system RAM available (default "/var/folders/lp/k0g2hcfs0bz1c4zn90pnh32w0000gn/T/gobonniegoParent543610505")
  -threads int
    	The number of concurrent readers/writers, defaults to the number of CPU cores (default 8)
  -size int
    	The aggregate size of disk test files in GiB, defaults to twice the physical RAM (default 128)
  -v	Verbose. Will print to stderr diagnostic information such as the amount of RAM, number of cores, etc.
  -version
    	Version. Will print the current version of gobonniego and then exit
```

## Technical Notes

`gobonniego` detects the number of CPU cores and the amount of RAM.

The number of cores may not match the number of physical cores. For example, an
Intel core i5 with two physical cores and hyperthreading is detected as 4
cores.

`gobonniego` spawns one thread for each core unless overridden by the `-threads`
flag.

`gobonniego` writes twice the amount of RAM unless overridden with the `-size`
flag.  For example, on a system with 16 GiB of RAM, `gobonniego` would write 32
GiB of data. This is to reduce the effect of the [buffer
cache](http://www.tldp.org/LDP/sag/html/buffer-cache.html), which may give
misleadingly good results.

If the sequential read performance is several multiples of the sequential write
performance, it's likely that the buffer cache has skewed the results.

`gobonniego` divides the total amount to write by the number of threads. For
example, a 4-core system with 8 GiB of RAM would have four threads each of
which would concurrently write 4 GiB of data for a total of 16 GiB.

`gobonniego` writes with buffered I/O; however, it waits for
[`bufio.Flush()`](https://golang.org/pkg/bufio/#Writer.Flush) to complete
before recording the duration.

`gobonniego` creates a 64 kiB chunk of random data which it writes in succession
to disk.  It's random in order to avoid inflating the results for filesystems
which enable compression (e.g. ZFS).

`gobonniego` reads the files concurrently in 64 kiB chunks. Every 127 chunks it
does a byte comparison against the original random data 64 kiB chunk to make
sure there has been no corruption. This probably exacts a small penalty in
read performance.

For IOPS measurement, a `gobonniego` thread seeks to a random position in the
file and reads 512 bytes. This counts as a single operation. Every tenth seek
instead of reading it will write 512 bytes of data. This also counts as an
operation. The ratio of reads:writes is 10:1, in order to approximate the ratio
that the TPC-E benchmark uses
(<http://www.cs.cmu.edu/~chensm/papers/TPCE-sigmod-record10.pdf>).

The IOPS measurement cycle runs for approximately 15 seconds, at the end of
which `gobonniego` tallies up the number of I/O operations and divides by the
duration of the test.

`gobonniego` uses
[`ioutil.TempDir()`](https://golang.org/pkg/io/ioutil/#TempDir) to create the
temporary directory in which to place its files, unless overridden by the
`-dir` flag. On Linux systems this temporary directory is often `/tmp/`, on
macOS systems, `/var/folders/...`.

## Bugs

If `gobonniego` fills up the filesystem, it will crash and you will need to find
& delete the `gobonniego` files manually. Below is a sample `find` command to
locate the `gobonniego` directory; delete that directory and everything
underneath:

```
find / -name gobonniegoParent\*
```

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

The impetus for writing `gobonniego` is to provide concurrency.  During a
benchmark of a ZFS filesystem (using *bonnie++*), it became clear that a the
single-threaded performance of *bonnie++* and not disk speed was the limiting
factor.
