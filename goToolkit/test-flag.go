package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var mu sync.RWMutex

func updateStatus(status string) {
	mu.RLock()
	defer mu.RUnlock()

}

func main() {
	// ...create abort channel...
	abort := make(chan struct{})

	go func() {
		os.Stdin.Read(make([]byte, 1)) // read a single byte
		abort <- struct{}{}
	}()

	fmt.Println("Commencing countdown.  Press return to abort.")
	for true {
		select {
		case <-time.After(3 * time.Second):
			launch()
		// Do nothing.
		case <-abort:
			fmt.Println("Launch aborted!")
			return
		default:
			fmt.Println("default case.  Launching...")
			time.Sleep(1 * time.Second)
		}
	}
}

func launch() {
	fmt.Println("rockets flying...  Boom!")
}
