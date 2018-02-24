package main

import (
	"fmt"
	"github.com/cunnie/gobonniego/mem"
	"log"
	"math/rand"
)

const Blocksize = 1 << 20 // 1 MiB

/*
 Balloon tries to consume enough memory to force the kernel to evict the buffer cache from RAM

 It's useful for disk benchmarks when you don't want the buffer cache to influence the results
*/

func main() {
	var balloon [][]byte
	physmem, err := mem.Get()
	check(err)

	for i := 0; i < int(physmem>>30); i++ {
		// fill up a GiB
		for j := 0; j < 1024; j++ {
			OneMiBBlock := make([]byte, Blocksize)
			lenRandom, err := rand.Read(OneMiBBlock)
			check(err)
			if lenRandom != Blocksize {
				panic(fmt.Sprintf("lenRandom %d is not equal to Blocksize %d", lenRandom, Blocksize))
			}
			balloon = append(balloon, OneMiBBlock)
		}
		log.Printf("GiB #%d", i)
	}
	for i := 0; i < 10; i++ {
		x := rand.Intn(len(balloon))
		y := rand.Intn(Blocksize)
		if balloon[x][y] == 'b' {
			log.Print("Today is your lucky day: balloon[x][y] equals 'b'")
		}
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
