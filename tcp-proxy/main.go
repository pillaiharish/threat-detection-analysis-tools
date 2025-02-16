package main

import (
	"log"
	"net"
	"os"
)

func handleConnection(client net.Conn, targetAddress string) {
	defer client.Close()

	server, err := net.Dial("tcp", targetAddress)
	if err != nil {
		log.Println("Failed to connect to target:", err)
		return
	}
	defer server.Close()

	log.Printf("New connection: %s -> %s", client.RemoteAddr(), targetAddress)

	// Capture and log traffic
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := client.Read(buf)
			if err != nil {
				break
			}
			log.Printf("[CLIENT] %s", string(buf[:n]))
			server.Write(buf[:n])
		}
	}()

	buf := make([]byte, 4096)
	for {
		n, err := server.Read(buf)
		if err != nil {
			break
		}
		log.Printf("[SERVER] %s", string(buf[:n]))
		client.Write(buf[:n])
	}
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s <local-port> <target-host:port>", os.Args[0])
	}

	localPort := os.Args[1]
	targetAddress := os.Args[2]

	listener, err := net.Listen("tcp", ":"+localPort)
	if err != nil {
		log.Fatal("Failed to start listener:", err)
	}
	defer listener.Close()

	log.Printf("TCP Proxy started. Listening on port %s and forwarding to %s", localPort, targetAddress)

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(client, targetAddress)
	}
}
