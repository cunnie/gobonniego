## bonniego

bonniego is a _minimal_ implementation of Tim Bray's
[bonnie](https://code.google.com/p/bonnie-64/) written in Go (*bonnie* is
written in C).

It measures disk throughput by reading and writing files.

It presents three disk metrics:

1. Sequential Write (higher is better)
2. Sequential Read (higher is better)
3. IOPS (I/O Operations per Second) (higher is better)

## Examples

The default settings are typically adequate; the program can be invoked without
parameters:

```
bonniego
```

Typical output:

```
Sequential Write MiB/s: 1229.10
Sequential Read MiB/s: 6729.41
IOPS: 26156
```

Running with the verbose option will print additional timestamped information
to STDERR:

```
bonniego -v
```

Yields:

```
2018/02/12 08:05:02 Bonnie working directory: /var/folders/zp/vmj1nyzj6p567k5syt3hvq3h0000gn/T/bonniegoParent649139571/bonniego
2018/02/12 08:05:02 Number of concurrent processes: 4
2018/02/12 08:05:02 Total System RAM (MiB): 16384
2018/02/12 08:05:03 Written (MiB): 1024
2018/02/12 08:05:03 Duration (seconds): 0.833131
Sequential Write MiB/s: 1229.10
2018/02/12 08:05:03 Read (MiB): 1024
2018/02/12 08:05:03 Duration (seconds): 0.152168
Sequential Read MiB/s: 6729.41
2018/02/12 08:05:13 operations 262144
2018/02/12 08:05:13 Duration (seconds): 10.022507
IOPS: 26156
```

You can tell `bonniego` where to place its test files. This is useful if the
default filesystem is too small or if you want to test a specific disk.
`bonniego` will clean up after itself, and will not delete the directory it's
told to run in (you can safely specify `/tmp` or `/` as the directory). Here
are some examples:

```
bonniego -dir D:\
bonniego -dir /tmp
bonniego -dir /zfs/tank
```

You may specify the number of threads (Goroutines) to run with the `-procs`
flag. In this example, we spawn 8 threads:

```
bonniego -procs 8
```

## Technical Notes

`bonniego` detects the number of CPU cores and the amount of RAM.

The number of cores may not match the number of physical cores. For example, an
Intel core i5 with two physical cores and hyperthreading is detected as 4
cores.

`bonniego` spawns one thread for each core unless overridden by the `-procs`
flag.

`bonniego` writes twice the amount of RAM in order to reduce the effect of
[buffer cache](http://www.tldp.org/LDP/sag/html/buffer-cache.html), which may
give misleadingly good results. For example, on a system with 16 GiB of RAM,
`bonniego` would write 32 GiB of data.

`bonniego` divides the total amount to write by the number of threads. For
example, a 4-core system with 8 GiB of RAM would have four threads each of
which would concurrently write 4 GiB of data for a total of 16 GiB.

`bonniego` creates a 64 kiB chunk of random data which it writes in succession
to disk.  It's random in order to avoid inflating the results for filesystems
which enable compression (e.g. ZFS).

`bonniego` reads the files concurrently in 64 kiB chunks. Every 127 chunks it
does a byte comparison against the original random data 64 kiB chunk to make
sure there has been no corruption. This probably exacts a small penalty in
read performance.

For IOPS measurement, a `bonniego` thread seeks to a random position in the
file and reads 512 bytes. This counts as a single operation. Every tenth seek
instead of reading it will write 512 bytes of data. This also counts as an
operation. The ratio of reads:writes is 10:1, in order to approximate the ratio
that the TPC-E benchmark uses
(<http://www.cs.cmu.edu/~chensm/papers/TPCE-sigmod-record10.pdf>).

`bonniego` uses [`ioutil.TempDir()`](https://golang.org/pkg/io/ioutil/#TempDir)
to create the temporary directory in which to place its files, unless
overridden by the `-dir` flag. On Linux systems this temporary directory is
under `/var`.

## Bugs

If bonniego fills up the filesystem, it will crash and you will need to find &
delete the `bonniego` files manually.  Look for the director that begins with
`bonniegoParent` and delete it and everything underneath.

### Acknowledgements

The impetus for writing it in Go is to provide concurrency.  During a benchmark
of a ZFS filesystem (using *bonnie++*, not *bonnie*), it became clear that a
the single-threaded performance of *bonnie++* and not the speed of the disks
were the limiting factor.
