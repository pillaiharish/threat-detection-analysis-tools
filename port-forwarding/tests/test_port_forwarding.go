package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Printf("Failed to listen on port 8081: %v", err)
		return
	}
	defer listener.Close()
	fmt.Println("Listening on port 8081")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v", err)
			continue
		}

		go func(c net.Conn) {
			defer c.Close()
			reader := bufio.NewReader(c)
			for {
				message, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				fmt.Printf("Received message: %s", message)
				c.Write([]byte("Message received.\n"))
			}
		}(conn)
	}
}
