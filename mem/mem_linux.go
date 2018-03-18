package mem

import (
	"github.com/cloudfoundry/gosigar"
	"os"
)

func Get() (uint64, error) {
	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.Total, nil
}

func ClearBufferCache() error {
	// don't need to do `sync` beforehand; the following drops only unused caches
	f, err := os.Create("/proc/sys/vm/drop_caches")
	if err != nil {
		return err
	}
	f.WriteString("3")
	f.Close()
	return nil
}
