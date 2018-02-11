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

	fileSize := int(mem.Total) * 2 / cores

	dir, err := ioutil.TempDir("", "bonniego")
	fmt.Printf("directory: %s\n", dir)
	check(err)
	defer os.RemoveAll(dir)

	randomBlock := make([]byte, Blocksize)
	_, err = rand.Read(randomBlock)
	check(err)

	start := time.Now()

	bytesReadorWritten := make(chan int)

	for i := 0; i < cores; i++ {
		go testWritePerformance(path.Join(dir, fmt.Sprintf("bonnie.%d", i)), fileSize, randomBlock, bytesReadorWritten)
	}
	bytesWritten := 0
	for i := 0; i < cores; i++ {
		bytesWritten += <-bytesReadorWritten
	}

	finish := time.Now()
	duration := finish.Sub(start)
	fmt.Printf("wrote %d MiB\n", bytesWritten>>20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))

	start = time.Now()

	for i := 0; i < cores; i++ {
		go testReadPerformance(path.Join(dir, fmt.Sprintf("bonnie.%d", i)), randomBlock, bytesReadorWritten)
	}
	bytesRead := 0
	for i := 0; i < cores; i++ {
		bytesRead += <-bytesReadorWritten
	}

	finish = time.Now()
	duration = finish.Sub(start)

	fmt.Printf("read %d MiB\n", bytesRead>>20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))
}

func testWritePerformance(filename string, fileSize int, randomBlock []byte, bytesWrittenChannel chan<- int) {
	f, err := os.Create(filename)
	check(err)
	defer f.Close()

	w := bufio.NewWriter(f)

	bytesWritten := 0
	for i := 0; i < fileSize; i += len(randomBlock) {
		n, err := w.Write(randomBlock)
		check(err)
		bytesWritten += n
	}

	w.Flush()
	f.Close()
	bytesWrittenChannel <- bytesWritten
}

func testReadPerformance(filename string, randomBlock []byte, bytesReadChannel chan<- int) {
	f, err := os.Open(filename)
	check(err)
	defer f.Close()

	bytesRead := 0
	data := make([]byte, Blocksize)

	for {
		n, err := f.Read(data)
		bytesRead += n
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if bytesRead%127 == 0 { // every hundredth or so block, do a sanity check. 127 is prime to avoid collisions
			if ! bytes.Equal(randomBlock, data) {
				panic("last block didn't match")
			}
		}
	}

	bytesReadChannel <- bytesRead
	f.Close()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
