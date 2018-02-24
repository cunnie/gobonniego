package bench

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"time"
)

const Blocksize = 0x1 << 16 // 65,536 bytes, 2^16 bytes

type Mark struct {
	BonnieDir                   string
	AggregateTestFilesSizeInGiB float64
	NumReadersWriters           int
	PhysicalMemory              uint64
	Result                      Result
	fileSize                    int
	randomBlock                 []byte
}

type Result struct {
	WrittenBytes    int
	WrittenDuration time.Duration
	ReadBytes       int
	ReadDuration    time.Duration
	IOPSOperations  int
	IOPSDuration    time.Duration
}

// the Sequential Write test must be called before the other two tests, for it creates the files
func (bm *Mark) RunSequentialWriteTest() {
	bm.fileSize = (int(bm.AggregateTestFilesSizeInGiB*(1<<10)) << 20) / bm.NumReadersWriters

	bytesWritten := make(chan int)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadWriteTest(path.Join(fmt.Sprintf("bonnie.%d", i)), bytesWritten)
	}
	bm.Result.WrittenBytes = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		bm.Result.WrittenBytes += <-bytesWritten
	}

	bm.Result.WrittenDuration = time.Now().Sub(start)
}

func (bm *Mark) RunSequentialReadTest() {
	bytesRead := make(chan int)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadReadTest(path.Join(fmt.Sprintf("bonnie.%d", i)), bytesRead)
	}
	bm.Result.ReadBytes = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		bm.Result.ReadBytes += <-bytesRead
	}

	bm.Result.ReadDuration = time.Now().Sub(start)
}

func (bm *Mark) RunIOPSTest() {
	opsPerformed := make(chan int)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadIOPSTest(path.Join(fmt.Sprintf("bonnie.%d", i)), opsPerformed)
	}
	bm.Result.IOPSOperations = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		bm.Result.IOPSOperations += <-opsPerformed
	}

	bm.Result.IOPSDuration = time.Now().Sub(start)
}

// calling program should `defer os.RemoveAll(bm.BonnieDir)` to clean up after run
func (bm *Mark) SetBonnieDir(parentDir string) error {
	// if bonnieParentDir exists, e.g. "/tmp", all is good, but if it doesn't, e.g. "/tmp/gobonniego_run_five", then create it
	fileInfo, err := os.Stat(parentDir)
	if err != nil {
		err = os.Mkdir(parentDir, 0755)
		if err != nil {
			return fmt.Errorf("SetBonnieDir(): %s", err)
		}
	}
	if !fileInfo.IsDir() {
		return errors.New(fmt.Sprintf("'%s' is not a directory!", parentDir))
	}
	bm.BonnieDir = path.Join(parentDir, "gobonniego")
	err = os.Mkdir(bm.BonnieDir, 0755)
	if err != nil {
		return fmt.Errorf("SetBonnieDir(): %s", err)
	}
	return nil
}

func (bm *Mark) CreateRandomBlock() error {
	// randomBlock has random data to prevent filesystems which use compression (e.g. ZFS) from having an unfair advantage
	bm.randomBlock = make([]byte, Blocksize)
	lenRandom, err := rand.Read(bm.randomBlock)
	if err != nil {
		return fmt.Errorf("CreateRandomBlock(): %s", err)
	}
	if len(bm.randomBlock) != lenRandom {
		return fmt.Errorf("CreateRandomBlock(): RandomBlock didn't get the correct number of bytes, %d != %d",
			len(bm.randomBlock), lenRandom)
	}
	return nil
}

func (bm *Mark) singleThreadWriteTest(filename string, bytesWrittenChannel chan<- int) {
	f, err := os.Create(path.Join(bm.BonnieDir, filename))
	check(err)
	defer f.Close()

	w := bufio.NewWriter(f)

	bytesWritten := 0
	for i := 0; i < bm.fileSize; i += len(bm.randomBlock) {
		n, err := w.Write(bm.randomBlock)
		check(err)
		bytesWritten += n
	}

	w.Flush()
	f.Close()
	bytesWrittenChannel <- bytesWritten
}

func (bm *Mark) singleThreadReadTest(filename string, bytesReadChannel chan<- int) {
	f, err := os.Open(path.Join(bm.BonnieDir, filename))
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
			if !bytes.Equal(bm.randomBlock, data) {
				panic("last block didn't match")
			}
		}
	}

	bytesReadChannel <- bytesRead
	f.Close()
}

func (bm *Mark) singleThreadIOPSTest(filename string, numOpsChannel chan<- int) {
	diskBlockSize := 0x1 << 9 // 512 bytes, nostalgia: in the olden (System V) days, disk blocks were 512 bytes
	fileInfo, err := os.Stat(path.Join(bm.BonnieDir, filename))
	check(err)
	fileSizeLessOneDiskBlock := fileInfo.Size() - int64(diskBlockSize) // give myself room to not read past EOF
	numOperations := 0

	f, err := os.OpenFile(path.Join(bm.BonnieDir, filename), os.O_RDWR, 0644)
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
		numOperations++
	}
	f.Close() // redundant, I know. I want to make sure writes are flushed
	numOpsChannel <- int(numOperations)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
