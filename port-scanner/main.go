package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

var address string = "192.168.0.126"
var startPort int = 0
var endPort int = 65535
var maxGoroutines = 100

func main() {
	activeThreads := 0 // to keep track of the number of active goroutines.

	// Buffered channel to signal the completion of port scanning tasks.
	doneChan := make(chan bool, endPort-startPort+1) // to keep track of completed ports

	// guard is buffered channel to limit the number of concurrent goroutines.
	guard := make(chan struct{}, maxGoroutines) // to make sure go routines donot exceed maxGoroutines

	for port := startPort; port <= endPort; port++ {
		guard <- struct{}{} // block if guard channel is full

		go func(port int) {
			defer func() { <-guard }() // release the slot in guard channel when done
			tcpPortScan(address, port, doneChan)
		}(port)
		activeThreads++
	}
	for activeThreads > 0 {
		<-doneChan
		activeThreads--
	}
	log.Println("Port scanning completed")
}

func tcpPortScan(address string, port int, doneChan chan bool) {
	_, err := net.DialTimeout("tcp", address+":"+strconv.Itoa(port), time.Second*10)
	if err == nil {
		fmt.Printf("Open port %d \n", port)
	}
	doneChan <- true
}
