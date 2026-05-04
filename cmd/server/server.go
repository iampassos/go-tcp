package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/iampassos/go-tcp/internal/tcp"
)

func main() {
	listener, err := tcp.Listen("8080")
	if err != nil {
		log.Fatalf("couldn't start server: %v", err)
	}
	defer listener.Close()

	log.Println("listening on port 8080")

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("Window size (1-5 default is 5): ")
	scanner.Scan()
	var size int
	fmt.Sscan(scanner.Text(), &size)
	if size == 0 {
		size = 5
	} else if size > 5 {
		size = 5
	} else if size < 0 {
		size = 1
	}

	for {
		connection, err := listener.Accept(size)
		if err != nil {
			log.Println("error accepting:", err)
			continue
		}

		go handleConnection(connection)
	}
}

func handleConnection(connection *tcp.Connection) {
	defer connection.Close()

	for {
		_, err := connection.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}

			log.Printf("error receiving: %v", err)
			return
		}
	}
}
