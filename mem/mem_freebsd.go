package mem

import (
	"bytes"
	"encoding/binary"
	"syscall"
	"unsafe"
)

func Get() (uint64, error) {
	var physMem uint64

	if err := sysctlbyname("hw.physmem", &physMem); err != nil {
		return 0, err
	}
	return physMem, nil
}

// generic Sysctl buffer unmarshalling
func sysctlbyname(name string, data interface{}) (err error) {
	val, err := syscall.Sysctl(name)
	if err != nil {
		return err
	}

	buf := []byte(val)

	switch v := data.(type) {
	case *uint64:
		*v = *(*uint64)(unsafe.Pointer(&buf[0]))
		return
	}

	bbuf := bytes.NewBuffer([]byte(val))
	return binary.Read(bbuf, binary.LittleEndian, data)
}
