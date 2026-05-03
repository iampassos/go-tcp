package tcp

import (
	"bytes"
	"encoding/gob"
	"net"
)

type ClientTransport struct {
	conn net.Conn
}

type ServerTransport struct {
	listener net.Listener
}

func InitClientTransport(addr string) (*ClientTransport, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	transport := &ClientTransport{conn: conn}

	return transport, nil
}

func (t *ClientTransport) Send(segment Segment) error {
	var buf bytes.Buffer

	err := gob.NewEncoder(&buf).Encode(segment)
	if err != nil {
		return err
	}

	_, err = t.conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (t *ClientTransport) Receive() (*Segment, error) {
	var segment Segment

	err := gob.NewDecoder(t.conn).Decode(&segment)
	if err != nil {
		return nil, err
	}

	return &segment, nil
}

func (t *ClientTransport) Close() error {
	err := t.conn.Close()
	if err != nil {
		return err
	}

	t.conn = nil

	return nil
}

func (t *ClientTransport) addr() string {
	return t.conn.RemoteAddr().String()
}

func InitServerTransport(port string) (*ServerTransport, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	transport := &ServerTransport{listener: listener}

	return transport, nil
}

func (t *ServerTransport) Accept() (ClientTransporter, error) {
	conn, err := t.listener.Accept()
	if err != nil {
		t.listener.Close()
		return nil, err
	}

	return &ClientTransport{conn: conn}, nil
}

func (t *ServerTransport) Close() error {
	err := t.listener.Close()
	if err != nil {
		return err
	}

	t.listener = nil

	return nil
}

func (t *ServerTransport) addr() string {
	return t.listener.Addr().String()
}
