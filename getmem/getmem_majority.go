// +build darwin linux windows

package getmem

import "github.com/cloudfoundry/gosigar"

func Getmem() (uint64, error) {
	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.Total, nil
}
