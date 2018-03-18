package mem

import (
	"errors"
	"github.com/cloudfoundry/gosigar"
	"runtime"
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
	return errors.New("Can't clear buffer cache; OS is \"" + runtime.GOOS + "\", not \"linux\" or \"darwin\"")
}
