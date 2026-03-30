package tcp

import (
	"bytes"
	"encoding/gob"
	"net"
)

type Transport struct {
	conn     net.Conn
	listener net.Listener
}

func InitClientTransport(addr string) (*Transport, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	transport := &Transport{conn: conn}

	return transport, nil
}

func InitServerTransport(port string) (*Transport, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	conn, err := listener.Accept()
	if err != nil {
		listener.Close()
		return nil, err
	}

	transport := &Transport{listener: listener, conn: conn}

	return transport, nil
}

func (t *Transport) Send(segment Segment) error {
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

func (t *Transport) Receive() (*Segment, error) {
	var segment Segment

	err := gob.NewDecoder(t.conn).Decode(&segment)
	if err != nil {
		return nil, err
	}

	return &segment, nil
}

func (t *Transport) Close() error {
	if t.conn != nil {
		err := t.conn.Close()
		if err != nil {
			return err
		}
		t.conn = nil
	}

	if t.listener != nil {
		err := t.listener.Close()
		if err != nil {
			return err
		}
		t.listener = nil
	}

	return nil
}

