## bonniego

bonniego is an implementation of Tim Bray's [bonnie](https://code.google.com/p/bonnie-64/) written in Go (*bonnie* is written in C).

It will show three disk metrics:

1. Sequential Write
2. Sequential Read
3. IOPS (I/O Operations per Second)

## Examples

```
bonniego
```

## Bugs

If the filesystem fills up while the program is running you may need to clean out the bonniego files manually.

### Acknowledgements

The impetus for writing it in Go is to provide concurrency.  During a benchmark of a ZFS filesystem (using *bonnie++*, not *bonnie*), it became clear that a the single-threaded performance of *bonnie++* and not the speed of the disks were the limiting factor
