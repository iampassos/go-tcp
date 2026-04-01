package main

import (
	"log"

	"github.com/iampassos/go-tcp/internal/tcp"
)

func main() {
	_, err := tcp.Dial("localhost:8080")
	if err != nil {
		log.Fatalf("couldn't connect to server: %v", err)
	}

	log.Println("Connected to server on address localhost:8080")
}
