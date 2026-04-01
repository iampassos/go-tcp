package tcp

import (
	"errors"
)

type Connection struct {
	State     State
	transport ClientTransporter
}

func Dial(addr string) (*Connection, error) {
	clientTransport, err := InitClientTransport(addr)
	if err != nil {
		return nil, err
	}

	connection := &Connection{State: CLOSED, transport: clientTransport}

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Syn: true}}, Message: &Message{Protocol: "gbn", MaxChars: 30}})
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

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Ack: true}}})
	if err != nil {
		return nil, err
	}

	return connection, nil
}

func (c *Connection) Close() error {
	return c.transport.Close()
}

