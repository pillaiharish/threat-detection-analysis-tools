package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var address string = "192.168.0.126"
var startPort int = 0
var endPort int = 65535
var maxGoroutines = 100

func main() {
	// Set up logging to output to a file and standard output
	logFile, err := os.OpenFile("port_scanner.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	log.Println("Starting port scan...")
	log.Printf("Scanning IP: %s, Ports: %d-%d, Max Goroutines: %d", address, startPort, endPort, maxGoroutines)

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
	startTime := time.Now()
	_, err := net.DialTimeout("tcp", address+":"+strconv.Itoa(port), time.Second*10)
	elapsedTime := time.Since(startTime)

	if err == nil {
		log.Printf("Open port %d found (Time: %s)\n", port, elapsedTime)
		fmt.Printf("Open port %d \n", port)
	} //else {
	// log.Printf("Port %d closed (Time: %s)\n", port, elapsedTime)
	// }

	doneChan <- true
}
