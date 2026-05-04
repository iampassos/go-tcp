package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/iampassos/go-tcp/internal/tcp"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Server ip (default is localhost:8080): ")
	scanner.Scan()
	server := scanner.Text()
	if server == "" {
		server = "localhost:8080"
	}

	fmt.Print("Protocol (gbn/sr default is gbn): ")
	scanner.Scan()
	protocolText := scanner.Text()
	protocol := tcp.Protocol(protocolText)
	if protocolText == "" {
		protocol = tcp.GoBackN
	}

	fmt.Print("Max chars (min 30 default is 30): ")
	scanner.Scan()
	var maxChars int
	fmt.Sscan(scanner.Text(), &maxChars)
	if maxChars == 0 {
		maxChars = 30
	}

	conn, err := tcp.Dial(server, protocol, maxChars)
	if err != nil {
		log.Fatalf("couldn't connect to server: %v", err)
	}
	defer conn.Close()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		text := scanner.Text()
		if text == "" {
			continue
		}

		if text == "exit" {
			break
		}

		err = conn.Send(text)
		if err != nil {
			log.Printf("error sending: %v", err)
			continue
		}
		conn.CloseWrite()
	}
}
