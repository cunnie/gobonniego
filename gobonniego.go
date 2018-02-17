package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/cunnie/gobonniego/getmem"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path"
	"runtime"
	"time"
)

const Version = "1.0.1"
const Blocksize = 0x1 << 16 // 65,536 bytes, 2^16 bytes

func main() {
	var bonnieTempDir, bonnieParentDir, bonnieDir string
	var numReadersWriters, aggregateTestFilesSizeInGiB int
	var verbose, version bool
	bonnieTempDir, err := ioutil.TempDir("", "gobonniegoParent")
	check(err)
	defer os.RemoveAll(bonnieTempDir)
	physicalMemory, err := getmem.Getmem()
	check(err)

	flag.BoolVar(&verbose, "v", false, "Verbose. Will print to stderr diagnostic information such as the amount of RAM, number of cores, etc.")
	flag.BoolVar(&version, "version", false, "Version. Will print the current version of gobonniego and then exit")
	flag.IntVar(&numReadersWriters, "threads", runtime.NumCPU(), "The number of concurrent readers/writers, defaults to the number of CPU cores")
	flag.IntVar(&aggregateTestFilesSizeInGiB, "size", 2*int(physicalMemory)>>30, "The aggregate size of disk test files in GiB, defaults to twice the physical RAM")
	flag.StringVar(&bonnieParentDir, "dir", bonnieTempDir, "The directory in which gobonniego places its temp files, should have at least twice system RAM available")
	flag.Parse()

	if version {
		fmt.Printf("gobonniego version %s\n", Version)
		os.Exit(0)
	}

	// if bonnieParentDir exists, e.g. "/tmp", all is good, but if it doesn't, e.g. "/tmp/gobonniego_run_five", then create it
	fileInfo, err := os.Stat(bonnieParentDir)
	if err != nil {
		err = os.Mkdir(bonnieParentDir, 0755)
		check(err)
		defer os.RemoveAll(bonnieParentDir)
	}
	if !fileInfo.IsDir() {
		panic(fmt.Sprintf("'%s' is not a directory!", bonnieParentDir))
	}

	bonnieDir = path.Join(bonnieParentDir, "gobonniego")
	err = os.Mkdir(bonnieDir, 0755)
	check(err)
	defer os.RemoveAll(bonnieDir)

	if verbose {
		log.Printf("Number of CPU cores: %d\n", runtime.NumCPU())
		log.Printf("Number of concurrent threads: %d\n", numReadersWriters)
		log.Printf("Total system RAM (MiB): %d\n", physicalMemory>>20)
		log.Printf("Amount of disk space to be used during test (MiB): %d\n", aggregateTestFilesSizeInGiB<<10)
		log.Printf("Bonnie working directory: %s\n", bonnieDir)
	}

	fileSize := (aggregateTestFilesSizeInGiB << 30) / numReadersWriters

	// randomBlock has random data to prevent filesystems which use compression (e.g. ZFS) from having an unfair advantage
	randomBlock := make([]byte, Blocksize)
	lenRandom, err := rand.Read(randomBlock)
	check(err)
	if len(randomBlock) != lenRandom {
		panic("RandomBlock didn't get the correct number of bytes")
	}

	start := time.Now()

	bytesReadorWritten := make(chan int)

	for i := 0; i < numReadersWriters; i++ {
		go testWritePerformance(path.Join(bonnieDir, fmt.Sprintf("bonnie.%d", i)), fileSize, randomBlock, bytesReadorWritten)
	}
	bytesWritten := 0
	for i := 0; i < numReadersWriters; i++ {
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

	for i := 0; i < numReadersWriters; i++ {
		go testReadPerformance(path.Join(bonnieDir, fmt.Sprintf("bonnie.%d", i)), randomBlock, bytesReadorWritten)
	}
	bytesRead := 0
	for i := 0; i < numReadersWriters; i++ {
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

	for i := 0; i < numReadersWriters; i++ {
		go testIOPerformance(path.Join(bonnieDir, fmt.Sprintf("bonnie.%d", i)), numOperationsChan)
	}
	numOperations := 0
	for i := 0; i < numReadersWriters; i++ {
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
			if !bytes.Equal(randomBlock, data) {
				panic("last block didn't match")
			}
		}
	}

	bytesReadChannel <- bytesRead
	f.Close()
}

func testIOPerformance(filename string, numOpsChannel chan<- int) {
	diskBlockSize := 0x1 << 9 // 512 bytes, nostalgia: in the olden (System V) days, disk blocks were 512 bytes
	fileInfo, err := os.Stat(filename)
	check(err)
	fileSizeLessOneDiskBlock := fileInfo.Size() - int64(diskBlockSize) // give myself room to not read past EOF
	numOperations := 0x1 << 16                                         // a fancy way of saying 65,536

	f, err := os.OpenFile(filename, os.O_RDWR, 0644)
	check(err)
	defer f.Close()

	data := make([]byte, diskBlockSize)
	checksum := make([]byte, diskBlockSize)

	start := time.Now()
	for i := 0; time.Now().Sub(start).Seconds() < 15.0; i++ { // run for 15 seconds then blow this taco joint
		f.Seek(rand.Int63n(fileSizeLessOneDiskBlock), 0)
		// TPC-E has a reads:writes ratio of 9.7:1  http://www.cs.cmu.edu/~chensm/papers/TPCE-sigmod-record10.pdf
		// we round to 10:1
		if i%10 != 0 {
			length, err := f.Read(data)
			check(err)
			if length != diskBlockSize {
				panic(fmt.Sprintf("I expected to read %d bytes, instead I read %d bytes!", diskBlockSize, length))
			}
			for j := 0; j < diskBlockSize; j++ {
				checksum[j] ^= data[j]
			}
		} else {
			length, err := f.Write(checksum)
			check(err)
			if length != diskBlockSize {
				panic(fmt.Sprintf("I expected to write %d bytes, instead I wrote %d bytes!", diskBlockSize, length))
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
