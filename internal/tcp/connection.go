package tcp

import (
	"errors"
	"log"
	"math/rand"
)

type Connection struct {
	ISN        int
	State      State
	transport  ClientTransporter
	Protocol   string
	MaxChars   int
	WindowSize int
	Seq        int
}

func Dial(addr string, message Message) (*Connection, error) {
	if message.MaxChars < 30 {
		return nil, errors.New("max chars must be at least 30")
	}

	if message.Protocol != "gbn" && message.Protocol != "sr" {
		return nil, errors.New("protocol must be either gbn or sr")
	}

	clientTransport, err := InitClientTransport(addr)
	if err != nil {
		return nil, err
	}

	connection := &Connection{State: CLOSED, transport: clientTransport, ISN: rand.Int()}

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Syn: true}, Seq: connection.ISN}, Message: message})
	if err != nil {
		return nil, err
	}

	connection.State = SYN_SENT

	segment, err := connection.transport.Receive()
	if err != nil {
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Syn || !segment.Header.Flags.Ack {
		return nil, errors.New("syn/ack flags not received")
	}

	connection.State = ESTABLISHED
	connection.Protocol = segment.Message.Protocol
	connection.MaxChars = segment.Message.MaxChars
	connection.WindowSize = segment.Header.WindowSize
	connection.Seq = connection.ISN + 1

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: segment.Header.Seq + 1, Seq: segment.Header.Ack}})
	if err != nil {
		return nil, err
	}

	log.Printf("[CLIENT] Connection established with %v. MaxChars: %v, Protocol: %v, WindowSize: %v", clientTransport.conn.RemoteAddr().String(), connection.MaxChars, connection.Protocol, connection.WindowSize)

	return connection, nil
}

func (c *Connection) Close() error {
	return c.transport.Close()
}
