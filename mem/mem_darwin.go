package mem

import (
	"github.com/cloudfoundry/gosigar"
	"os/exec"
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
	return exec.Command("/usr/sbin/purge").Run()
}
