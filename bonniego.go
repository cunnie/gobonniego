package main

import (
	"flag"
	"fmt"
	"time"
)

func main() {
	var ip = flag.Int("flagname", 1234, "help message for flagname")
	flag.Parse()
	fmt.Printf("ip: %d\n", *ip)
	PrintResults("Hello, world. The time is")
}

func PrintResults(results string) {
	var now = time.Now()
	fmt.Printf("%s, %d\n", results, now.UnixNano())
	// results
}
