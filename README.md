## bonniego

bonniego is a _minimal_ implementation of Tim Bray's [bonnie](https://code.google.com/p/bonnie-64/) written in Go (*bonnie* is written in C).

It will show three disk metrics:

1. Sequential Write
2. Sequential Read
3. IOPS (I/O Operations per Second)

## Examples

Default:
```
bonniego
```
Yields:
```
Sequential Write MiB/s: 1229.10
Sequential Read MiB/s: 6729.41
IOPS: 26156
```

Running with the verbose option will print additional timestamped information to STDERR:
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

You can tell `bonniego` where to place its test files. This is useful if the default filesystem is too small or
if you want to test a specific filesystem. `bonniego` will clean up after itself, and will not delete the directory
it's told to run in (you can safely specify `/tmp` or `/` as the directory). Here are some examples:
```
bonniego -dir D:\
bonniego -dir /tmp
bonniego -dir /zfs/tank
```

You tell `bonniego` how many threads to run. `bonniego` defaults to running the same number of threads as the number of
cores it detects (an Intel core i5 with two physical cores and hyperthreading enabled results in 4 virtual cores;
`bonniego` would run 4 threads by default on such a machine).

```
bonniego -procs 8
```

## Bugs

If bonniego fills up the filesystem, it will crash and you will need to find & delete the `bonniego` files manually.

### Acknowledgements

The impetus for writing it in Go is to provide concurrency.  During a benchmark of a ZFS filesystem (using *bonnie++*, not *bonnie*), it became clear that a the single-threaded performance of *bonnie++* and not the speed of the disks were the limiting factor
