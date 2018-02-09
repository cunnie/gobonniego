package main

import (
	"fmt"
	"time"
	"os"
	"bufio"
	"crypto/rand"
	"math"
)

func main() {
	//var ip = flag.Int("flagname", 1234, "help message for flagname")
	//flag.Parse()
	//fmt.Printf("ip: %d\n", *ip)


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
	start := time.Now()

	n, err := w.Write(b)
	if err != nil {
		panic("couldn't write")
	}

	w.Flush()
	finish := time.Now()
	duration := finish.Sub(start)
	fmt.Printf("wrote %d bytes\n", n)
	fmt.Printf("took %d nanoseconds\n", duration.Nanoseconds())
	fmt.Printf("throughput %0.2f MiB/s\n", float64(len(b))/float64(duration.Seconds())/math.Exp2(20))
}
