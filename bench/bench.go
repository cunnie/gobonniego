package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cunnie/gobonniego/mem"
	"io"
	"math/rand"
	"os"
	"path"
	"time"
)

const Version = "1.0.9"
const Blocksize = 0x1 << 16 // 65,536 bytes, 2^16 bytes

// bench.Mark{} -- haha! Get it? "benchmark"!
type Mark struct {
	Start                       time.Time `json:"start_time"`
	BonnieDir                   string    `json:"gobonniego_directory"`
	AggregateTestFilesSizeInGiB float64   `json:"disk_space_used_gib"`
	NumReadersWriters           int       `json:"num_readers_and_writers"`
	PhysicalMemory              uint64    `json:"physical_memory_bytes"`
	IODuration                  float64   `json:"iops_duration_seconds"`
	Results                     []Result  `json:"results"`
	fileSize                    int
	randomBlock                 []byte
}

type Result struct {
	Start           time.Time     `json:"start_time"`
	WrittenBytes    int           `json:"write_bytes"`
	WrittenDuration time.Duration `json:"write_seconds"`
	ReadBytes       int           `json:"read_bytes"`
	ReadDuration    time.Duration `json:"read_seconds"`
	IOOperations    int           `json:"io_operations"`
	IODuration      time.Duration `json:"io_seconds"`
}

type ThreadResult struct {
	Result int // bytes written, bytes read, or number of I/O operations
	Error  error
}

func (bm *Mark) Version() string {
	return Version
}

// the Sequential Write test must be called before the other two tests, for it creates the files
func (bm *Mark) RunSequentialWriteTest() error {
	bm.fileSize = (int(bm.AggregateTestFilesSizeInGiB*(1<<10)) << 20) / bm.NumReadersWriters

	// Delete pre-existing files, should only be necessary when there are two or more test runs
	for i := 0; i < bm.NumReadersWriters; i++ {
		err := os.RemoveAll(path.Join(bm.BonnieDir, fmt.Sprintf("bonnie.%d", i)))
		if err != nil {
			return err
		}
	}

	bm.Results = append(bm.Results, Result{Start: time.Now()}) // store new result
	newResult := &bm.Results[len(bm.Results)-1]

	bytesWritten := make(chan ThreadResult)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadWriteTest(path.Join(bm.BonnieDir, fmt.Sprintf("bonnie.%d", i)), bytesWritten)
	}
	newResult.WrittenBytes = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		result := <-bytesWritten
		if result.Error != nil {
			return result.Error
		}
		newResult.WrittenBytes += result.Result
	}

	newResult.WrittenDuration = time.Now().Sub(start)
	return nil
}

// ReadTest must be called before IOPSTest otherwise
// ReadTest will mistake IOPSTest's random writes for file corruption
func (bm *Mark) RunSequentialReadTest() error {
	newResult := &bm.Results[len(bm.Results)-1]
	mem.ClearBufferCache() // ignore errors; if it works great, if it doesn't, too bad

	bytesRead := make(chan ThreadResult)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadReadTest(path.Join(bm.BonnieDir, fmt.Sprintf("bonnie.%d", i)), bytesRead)
	}
	newResult.ReadBytes = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		result := <-bytesRead
		if result.Error != nil {
			return result.Error
		}
		newResult.ReadBytes += result.Result
	}

	newResult.ReadDuration = time.Now().Sub(start)
	return nil
}

func (bm *Mark) RunIOPSTest() error {
	newResult := &bm.Results[len(bm.Results)-1]
	mem.ClearBufferCache() // ignore errors; if it works great, if it doesn't, too bad

	opsPerformed := make(chan ThreadResult)
	start := time.Now()

	for i := 0; i < bm.NumReadersWriters; i++ {
		go bm.singleThreadIOPSTest(path.Join(bm.BonnieDir, fmt.Sprintf("bonnie.%d", i)), opsPerformed)
	}
	newResult.IOOperations = 0
	for i := 0; i < bm.NumReadersWriters; i++ {
		result := <-opsPerformed
		if result.Error != nil {
			return result.Error
		}
		newResult.IOOperations += result.Result
	}

	newResult.IODuration = time.Now().Sub(start)
	return nil
}

// Thanks Choly! http://choly.ca/post/go-json-marshalling/
func (bm Mark) MarshalJSON() ([]byte, error) {
	type Alias Mark
	return json.Marshal(&struct {
		Version string `json:"version"`
		Alias
	}{
		Version: Version,
		Alias:   Alias(bm),
	})
}

func (r Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		WriteMBps       float64 `json:"write_megabytes_per_second"`
		ReadMBps        float64 `json:"read_megabytes_per_second"`
		IOPS            float64 `json:"iops"`
		WrittenDuration float64 `json:"write_seconds"`
		ReadDuration    float64 `json:"read_seconds"`
		IODuration      float64 `json:"io_seconds"`
		Alias
	}{
		WriteMBps:       MegaBytesPerSecond(r.WrittenBytes, r.WrittenDuration),
		ReadMBps:        MegaBytesPerSecond(r.ReadBytes, r.ReadDuration),
		IOPS:            IOPS(r.IOOperations, r.IODuration),
		WrittenDuration: r.WrittenDuration.Seconds(),
		ReadDuration:    r.ReadDuration.Seconds(),
		IODuration:      r.IODuration.Seconds(),
		Alias:           Alias(r),
	})
}

// the following should be run as a goroutine as part of the benchmark; clears the buffer cache every 3 seconds
func ClearBufferCacheEveryThreeSeconds() error {
	for ; ; <-time.After(3 * time.Second) {
		err := mem.ClearBufferCache()
		if err != nil {
			return fmt.Errorf("Couldn't clear the buffer cache, bailing: %s", err)
		}
	}
}

func MegaBytesPerSecond(bytes int, duration time.Duration) float64 {
	return float64(bytes) / float64(duration.Seconds()) / 1000000
}

func IOPS(operations int, duration time.Duration) float64 {
	return float64(operations) / float64(duration.Seconds())
}

// calling program should `defer os.RemoveAll(bm.BonnieDir)` to clean up after run
func (bm *Mark) SetBonnieDir(parentDir string) error {
	// if bonnieParentDir exists, e.g. "/tmp", all is good, but if it doesn't, e.g. "/tmp/gobonniego_run_five", then create it
	err := createDirIfNeeded(parentDir)
	if err != nil {
		return err
	}
	bm.BonnieDir = path.Join(parentDir, "gobonniego")
	return createDirIfNeeded(bm.BonnieDir)
}

func createDirIfNeeded(dir string) error {
	fileInfo, err := os.Stat(dir)
	if err != nil {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	} else if !fileInfo.IsDir() {
		return errors.New(fmt.Sprintf("'%s' is not a directory!", dir))
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
	f, err := os.Create(filename)
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
	f, err := os.Open(filename)
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
	fileInfo, err := os.Stat(filename)
	if err != nil {
		numOpsChannel <- ThreadResult{
			Result: 0, Error: err,
		}
		return
	}
	fileSizeLessOneDiskBlock := fileInfo.Size() - int64(diskBlockSize) // give myself room to not read past EOF
	numOperations := 0

	f, err := os.OpenFile(filename, os.O_RDWR, 0644)
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
	for i := 0; time.Now().Sub(start).Seconds() < bm.IODuration; i++ { // run for xx seconds then blow this taco stand
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
