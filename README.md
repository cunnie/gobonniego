## bonniego

bonniego is an implementation of Tim Bray's [bonnie](https://code.google.com/p/bonnie-64/) written in Go (*bonnie* is written in C).


### Acknowledgements

The impetus for writing it in Go is to provide concurrency.  During a benchmark of a ZFS filesystem (using *bonnie++*, not *bonnie*), it became clear that a the single-threaded performance of *bonnie++* and not the speed of the disks were the limiting factor
