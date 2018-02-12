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
	"log"
)

const Blocksize = 0x1 << 16 // 65,536 bytes, 2^16 bytes

func main() {
	var bonnieTempDir, bonnieParentDir, bonnieDir string
	var numProcs int
	var verbose bool
	bonnieTempDir, err := ioutil.TempDir("", "bonniegoParent")
	check(err)
	defer os.RemoveAll(bonnieTempDir)

	flag.BoolVar(&verbose, "v", false, "Verbose. Will print to stderr diagnostic information such as the amount of RAM, number of cores, etc.")
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

	mem := sigar.Mem{}
	mem.Get()
	if verbose {
		log.Printf("Bonnie working directory: %s\n", bonnieDir)
		log.Printf("Number of concurrent processes: %d\n", numProcs)
		log.Printf("Total System RAM (MiB): %d\n", mem.Total>>20)
	}

	fileSize := int(mem.Total) * 2 / numProcs
	//fileSize = fileSize >> 5 // fixme: comment-out before committing. during testing; speeds up tests thirty-two-fold

	// randomBlock has random data to prevent filesystems which use compression (e.g. ZFS) from having an unfair advantage
	// we're testing hardware throughput, not filesystem throughput
	randomBlock := make([]byte, Blocksize)
	lenRandom, err := rand.Read(randomBlock)
	check(err)
	if len(randomBlock) != lenRandom {
		panic("RandomBlock didn't get the correct number of bytes")
	}

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
	if verbose {
		log.Printf("Written (MiB): %d\n", bytesWritten>>20)
		log.Printf("Duration (seconds): %f\n", duration.Seconds())
	}
	fmt.Printf("Sequential Write MiB/s: %0.2f\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))

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

	if verbose {
		log.Printf("Read (MiB): %d\n", bytesRead>>20)
		log.Printf("Duration (seconds): %f\n", duration.Seconds())
	}
	fmt.Printf("Sequential Read MiB/s: %0.2f\n", float64(bytesWritten)/float64(duration.Seconds())/math.Exp2(20))

	numOperationsChan := make(chan int)
	start = time.Now()

	for i := 0; i < numProcs; i++ {
		go testIOPerformance(path.Join(bonnieDir, fmt.Sprintf("bonnie.%d", i)), numOperationsChan)
	}
	numOperations := 0
	for i := 0; i < numProcs; i++ {
		numOperations += <-numOperationsChan
	}

	finish = time.Now()
	duration = finish.Sub(start)

	if verbose {
		log.Printf("operations %d\n", numOperations)
		log.Printf("Duration (seconds): %f\n", duration.Seconds())
	}
	fmt.Printf("IOPS: %0.0f\n", float64(numOperations)/float64(duration.Seconds()))
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
		// once every 127 blocks, do a sanity check. 127 is prime to avoid collisions
		// e.g. 128 would check every block, not every 128th block
		if bytesRead%127 == 0 {
			if ! bytes.Equal(randomBlock, data) {
				panic("last block didn't match")
			}
		}
	}

	bytesReadChannel <- bytesRead
	f.Close()
}

func testIOPerformance(filename string, numOpsChannel chan<- int) {
	diskBlock := 512 // in the olden days, disk blocks were 512 bytes
	fileInfo, err := os.Stat(filename)
	check(err)
	fileSize := fileInfo.Size() - int64(diskBlock) // give myself room to not read past EOF
	numOperations := 0x1 << 16                     // a fancy way of saying 65,536

	f, err := os.OpenFile(filename, os.O_RDWR, 0644)
	check(err)
	defer f.Close()

	data := make([]byte, diskBlock)
	checksum := make([]byte, diskBlock)

	for i := 0; i < numOperations; i++ {
		f.Seek(rand.Int63n(fileSize), 0)
		// TPC-E has a reads:writes ratio of 9.7:1  http://www.cs.cmu.edu/~chensm/papers/TPCE-sigmod-record10.pdf
		// we round to 10:1
		if i%10 != 0 {
			length, err := f.Read(data)
			check(err)
			if length != diskBlock {
				panic(fmt.Sprintf("I expected to read %d bytes, instead I read %d bytes!", diskBlock, length))
			}
			for i := 0; i < diskBlock; i++ {
				checksum[i] ^= data[i]
			}
		} else {
			length, err := f.Write(checksum)
			check(err)
			if length != diskBlock {
				panic(fmt.Sprintf("I expected to write %d bytes, instead I wrote %d bytes!", diskBlock, length))
			}
		}
	}
	numOpsChannel <- int(numOperations)
	f.Close()
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
