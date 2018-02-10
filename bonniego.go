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

	for i := 0; i < int(mem.Total>>4); i += len(randomBlock) { // fixme remove ">> 4"
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

	r := bufio.NewReader(f)
	bytesRead := 0
	readBuf := []byte{}

	rando := rand.New(rand.NewSource(time.Now().UnixNano()))
	var offset int
	start = time.Now()
	for n, err := r.Read(readBuf); n > 0; bytesRead += n {
		check(err)
		// let's do the comparison
		offset = rando.Int() % Blocksize
		if readBuf[offset] != randomBlock[offset] {
			panic("they didn't match")
		}
	}
	f.Close()
	finish = time.Now()

	duration = finish.Sub(start)
	fmt.Printf("read %d MiB\n", bytesRead>>20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
