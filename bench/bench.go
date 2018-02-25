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

type ThreadResult struct {
	Result int // bytes written, bytes read, or number of I/O operations
	Error  error
}

// the Sequential Write test must be called before the other two tests, for it creates the files
func (bm *Mark) RunSequentialWriteTest() error {
	bm.fileSize = (int(bm.AggregateTestFilesSizeInGiB*(1<<10)) << 20) / bm.NumReadersWriters

	bytesWritten := make(chan ThreadResult)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadWriteTest(path.Join(fmt.Sprintf("bonnie.%d", i)), bytesWritten)
	}
	bm.Result.WrittenBytes = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		result := <-bytesWritten
		if result.Error != nil {
			return result.Error
		}
		bm.Result.WrittenBytes += result.Result
	}

	bm.Result.WrittenDuration = time.Now().Sub(start)
	return nil
}

func (bm *Mark) RunSequentialReadTest() error {
	bytesRead := make(chan ThreadResult)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadReadTest(path.Join(fmt.Sprintf("bonnie.%d", i)), bytesRead)
	}
	bm.Result.ReadBytes = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		result := <-bytesRead
		if result.Error != nil {
			return result.Error
		}
		bm.Result.ReadBytes += result.Result
	}

	bm.Result.ReadDuration = time.Now().Sub(start)
	return nil
}

func (bm *Mark) RunIOPSTest() error {
	opsPerformed := make(chan ThreadResult)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadIOPSTest(path.Join(fmt.Sprintf("bonnie.%d", i)), opsPerformed)
	}
	bm.Result.IOPSOperations = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		result := <-opsPerformed
		if result.Error != nil {
			return result.Error
		}
		bm.Result.IOPSOperations += result.Result
	}

	bm.Result.IOPSDuration = time.Now().Sub(start)
	return nil
}

// calling program should `defer os.RemoveAll(bm.BonnieDir)` to clean up after run
func (bm *Mark) SetBonnieDir(parentDir string) error {
	// if bonnieParentDir exists, e.g. "/tmp", all is good, but if it doesn't, e.g. "/tmp/gobonniego_run_five", then create it
	fileInfo, err := os.Stat(parentDir)
	if err != nil {
		err = os.Mkdir(parentDir, 0755)
		if err != nil {
			return err
		}
	}
	if !fileInfo.IsDir() {
		return errors.New(fmt.Sprintf("'%s' is not a directory!", parentDir))
	}
	bm.BonnieDir = path.Join(parentDir, "gobonniego")
	err = os.Mkdir(bm.BonnieDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func (bm *Mark) CreateRandomBlock() error {
	// randomBlock has random data to prevent filesystems which use compression (e.g. ZFS) from having an unfair advantage
	bm.randomBlock = make([]byte, Blocksize)
	lenRandom, err := rand.Read(bm.randomBlock)
	if err != nil {
		return err
	}
	if len(bm.randomBlock) != lenRandom {
		return fmt.Errorf("CreateRandomBlock(): RandomBlock didn't get the correct number of bytes, %d != %d",
			len(bm.randomBlock), lenRandom)
	}
	return nil
}

func (bm *Mark) singleThreadWriteTest(filename string, bytesWrittenChannel chan<- ThreadResult) {
	f, err := os.Create(path.Join(bm.BonnieDir, filename))
	if err != nil {
		bytesWrittenChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	bytesWritten := 0
	for i := 0; i < bm.fileSize; i += len(bm.randomBlock) {
		n, err := w.Write(bm.randomBlock)
		if err != nil {
			bytesWrittenChannel <- ThreadResult{
				Result: 0, Error: err,
			}
			return
		}
		bytesWritten += n
	}

	err = w.Flush()
	if err != nil {
		bytesWrittenChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	err = f.Close()
	if err != nil {
		bytesWrittenChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	bytesWrittenChannel <- ThreadResult{Result: bytesWritten, Error: nil}
}

func (bm *Mark) singleThreadReadTest(filename string, bytesReadChannel chan<- ThreadResult) {
	f, err := os.Open(path.Join(bm.BonnieDir, filename))
	if err != nil {
		bytesReadChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	defer f.Close()

	bytesRead := 0
	data := make([]byte, Blocksize)

	for {
		n, err := f.Read(data)
		if err != nil {
			if err == io.EOF {
				break
			}
			bytesReadChannel <- ThreadResult{
				Result: 0, Error: err,
			}
			return
		}
		bytesRead += n
		// once every 127 blocks, do a sanity check. 127 is prime to avoid collisions
		// e.g. 128 would check every block, not every 128th block
		if bytesRead%127 == 0 {
			if !bytes.Equal(bm.randomBlock, data) {
				bytesReadChannel <- ThreadResult{
					Result: 0, Error: fmt.Errorf(
						"Most recent block didn't match random block, bytes read (includes corruption): %d",
						bytesRead),
				}
				return
			}
		}
	}

	bytesReadChannel <- ThreadResult{Result: bytesRead, Error: nil}
}

func (bm *Mark) singleThreadIOPSTest(filename string, numOpsChannel chan<- ThreadResult) {
	diskBlockSize := 0x1 << 9 // 512 bytes, nostalgia: in the olden (System V) days, disk blocks were 512 bytes
	fileInfo, err := os.Stat(path.Join(bm.BonnieDir, filename))
	if err != nil {
		numOpsChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	fileSizeLessOneDiskBlock := fileInfo.Size() - int64(diskBlockSize) // give myself room to not read past EOF
	numOperations := 0

	f, err := os.OpenFile(path.Join(bm.BonnieDir, filename), os.O_RDWR, 0644)
	if err != nil {
		numOpsChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	defer f.Close()

	data := make([]byte, diskBlockSize)
	checksum := make([]byte, diskBlockSize)

	start := time.Now()
	for i := 0; time.Now().Sub(start).Seconds() < 15.0; i++ { // run for 15 seconds then blow this taco stand
		f.Seek(rand.Int63n(fileSizeLessOneDiskBlock), 0)
		// TPC-E has a reads:writes ratio of 9.7:1  http://www.cs.cmu.edu/~chensm/papers/TPCE-sigmod-record10.pdf
		// we round to 10:1
		if i%10 != 0 {
			length, err := f.Read(data)
			if err != nil {
				numOpsChannel <- ThreadResult{
					Result: 0, Error: err,
				}
				return
			}
			if length != diskBlockSize {
				panic(fmt.Sprintf("I expected to read %d bytes, instead I read %d bytes!", diskBlockSize, length))
			}
			for j := 0; j < diskBlockSize; j++ {
				checksum[j] ^= data[j]
			}
		} else {
			length, err := f.Write(checksum)
			if err != nil {
				numOpsChannel <- ThreadResult{
					Result: 0, Error: err,
				}
				return
			}
			if length != diskBlockSize {
				numOpsChannel <- ThreadResult{
					Result: 0,
					Error: fmt.Errorf("I expected to write %d bytes, instead I wrote %d bytes!",
						diskBlockSize, length),
				}
				return
			}
		}
		numOperations++
	}
	err = f.Close() // redundant, I know. I want to make sure writes are flushed
	if err != nil {
		numOpsChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	numOpsChannel <- ThreadResult{Result: int(numOperations), Error: nil}
}
