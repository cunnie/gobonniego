// +build darwin linux windows

package mem

import "github.com/cloudfoundry/gosigar"

func Get() (uint64, error) {
	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return 0, err
	}
	return mem.Total, nil
}
