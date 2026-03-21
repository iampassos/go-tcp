package main

import (
	"log"

	"github.com/iampassos/go-tcp/internal/server"
)

func main() {
	sv := server.NewServer()
	err := sv.Start()
	if err != nil {
		log.Fatalf("couldn't start server: %v\n", err)
	}
}
