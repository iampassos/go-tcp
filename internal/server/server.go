package server

import (
	"log"
	"net"
)

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", ":5050")
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Println("started listening on :5050")

	for {
		_, err := listener.Accept()
		if err != nil {
			return err
		}
	}
}
