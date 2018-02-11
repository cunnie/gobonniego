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
	"flag"
)

const Blocksize = 0x1 << 16 // 65,536, 2^16

func main() {
	var bonnieTempDir, bonnieParentDir, bonnieDir string
	var numProcs int
	bonnieTempDir, err := ioutil.TempDir("", "bonniegoParent")
	check(err)
	defer os.RemoveAll(bonnieTempDir)

	flag.IntVar(&numProcs, "procs", runtime.NumCPU(), "The number of concurrent readers/writers, defaults to the number of CPU cores")
	flag.StringVar(&bonnieParentDir, "dir", bonnieTempDir, "The directory in which bonniego places its temp files, should have at least twice system RAM available")
	flag.Parse()

	// if bonnieParentDir exists, e.g. "/tmp", all is good, but if it doesn't, e.g. "/tmp/bonniego_run_five", then create it
	fileInfo, err := os.Stat(bonnieParentDir)
	if err != nil {
		err = os.Mkdir(bonnieParentDir, 0755)
		check(err)
		defer os.RemoveAll(bonnieParentDir)
	}
	if ! fileInfo.IsDir() {
		panic(fmt.Sprintf("'%s' is not a directory!", bonnieParentDir))
	}

	bonnieDir = path.Join(bonnieParentDir, "bonniego")
	err = os.Mkdir(bonnieDir, 0755)
	check(err)
	defer os.RemoveAll(bonnieDir)
	fmt.Printf("directory: %s\n", bonnieDir)

	fmt.Printf("cores: %d\n", numProcs)

	mem := sigar.Mem{}
	mem.Get()
	fmt.Printf("memory: %d MiB\n", mem.Total>>20)

	fileSize := int(mem.Total) * 2 / numProcs
	//fileSize = fileSize >> 4 // uncomment during testing; speeds up tests sixteen-fold

	randomBlock := make([]byte, Blocksize)
	_, err = rand.Read(randomBlock)
	check(err)

	start := time.Now()

	bytesReadorWritten := make(chan int)

	for i := 0; i < numProcs; i++ {
		go testWritePerformance(path.Join(bonnieDir, fmt.Sprintf("bonnie.%d", i)), fileSize, randomBlock, bytesReadorWritten)
	}
	bytesWritten := 0
	for i := 0; i < numProcs; i++ {
		bytesWritten += <-bytesReadorWritten
	}

	finish := time.Now()
	duration := finish.Sub(start)
	fmt.Printf("wrote %d MiB\n", bytesWritten>>20)
	fmt.Printf("took %f seconds\n", duration.Seconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))

	start = time.Now()

	for i := 0; i < numProcs; i++ {
		go testReadPerformance(path.Join(bonnieDir, fmt.Sprintf("bonnie.%d", i)), randomBlock, bytesReadorWritten)
	}
	bytesRead := 0
	for i := 0; i < numProcs; i++ {
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
