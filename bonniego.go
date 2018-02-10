package main

import (
	"fmt"
	"time"
	"os"
	"bufio"
	"math/rand"
	"math"
	"runtime"
	"github.com/cloudfoundry/gosigar"
	"io/ioutil"
	"path"
	"io"
	"bytes"
)

const Blocksize = 0x1 << 16 // 65,536, 2^16

func main() {

	//var ip = flag.Int("flagname", 1234, "help message for flagname")
	//flag.Parse()
	//fmt.Printf("ip: %d\n", *ip)

	cores := runtime.NumCPU()
	fmt.Printf("cores: %d\n", cores)

	mem := sigar.Mem{}
	mem.Get()
	fmt.Printf("memory: %d MiB\n", mem.Total>>20)

	dir, err := ioutil.TempDir("", "bonniego")
	check(err)
	defer os.RemoveAll(dir)

	f, err := os.Create(path.Join(dir, "bonniego"))
	check(err)
	defer f.Close()
	fmt.Printf("directory: %s\n", dir)

	randomBlock := make([]byte, Blocksize)
	_, err = rand.Read(randomBlock)
	check(err)

	w := bufio.NewWriter(f)
	bytesWritten := 0
	start := time.Now()

	for i := 0; i < int(mem.Total); i += len(randomBlock) { // fixme remove ">> 4"
		n, err := w.Write(randomBlock)
		bytesWritten += n
		check(err)
	}

	w.Flush()
	f.Close()
	finish := time.Now()
	duration := finish.Sub(start)
	fmt.Printf("wrote %d MiB\n", bytesWritten>>20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))

	f, err = os.Open(path.Join(dir, "bonniego"))
	check(err)
	defer f.Close()

	bytesRead := 0
	data := make([]byte, Blocksize)

	start = time.Now()

	for {
		n, err := f.Read(data)
		bytesRead += n
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			return
		}
	}

	finish = time.Now()
	duration = finish.Sub(start)

	fmt.Printf("read %d MiB\n", bytesRead >> 20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))

	if ! bytes.Equal(randomBlock, data) {
		panic("last block didn't match")
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
