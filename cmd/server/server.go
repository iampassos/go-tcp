package main

import (
	"log"

	"github.com/iampassos/go-tcp/internal/tcp"
)

func main() {
	listener, err := tcp.Listen("8080")
	if err != nil {
		log.Fatalf("Couldn't start server: %v", err)
	}
	defer listener.Close()

	log.Println("Listening on port 8080")

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting:", err)
			continue
		}

		go handleConnection(connection)
	}
}

func handleConnection(connection *tcp.Connection) {
	defer connection.Close()
}
