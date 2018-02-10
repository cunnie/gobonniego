package main

import (
	"fmt"
	"time"
	"os"
	"bufio"
	"crypto/rand"
	"math"
	"runtime"
	"github.com/cloudfoundry/gosigar"
)

func main() {
	//var ip = flag.Int("flagname", 1234, "help message for flagname")
	//flag.Parse()
	//fmt.Printf("ip: %d\n", *ip)

	cores := runtime.NumCPU()
	fmt.Printf("cores: %d\n", cores)

	mem := sigar.Mem{}
	mem.Get()
	fmt.Printf("memory: %d GiB\n", mem.Total>>30)

	f, err := os.Create("/tmp/dat2")
	if err != nil {
		panic("couldn't open")
	}
	defer f.Close()

	b := make([]byte, 0x1<<16)
	_, err = rand.Read(b)
	if err != nil {
		panic("couldn't rand")
	}

	w := bufio.NewWriter(f)
	bytesWritten := 0
	start := time.Now()

	for i := 0; i < int(mem.Total); i += len(b) {
		n, err := w.Write(b)
		bytesWritten += n
		if err != nil {
			panic("couldn't write")
		}
	}

	w.Flush()
	finish := time.Now()
	duration := finish.Sub(start)
	fmt.Printf("wrote %d MiB\n", bytesWritten>>20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))
}
