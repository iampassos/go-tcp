package tcp

import (
	"errors"
)

type Listener struct {
	transport ServerTransporter
}

func Listen(port string) (*Listener, error) {
	transporter, err := InitServerTransport(port)
	if err != nil {
		return nil, err
	}

	listener := &Listener{transport: transporter}

	return listener, nil
}

func (l *Listener) Accept() (*Connection, error) {
	clientTransport, err := l.transport.Accept()
	if err != nil {
		return nil, err
	}

	connection := &Connection{State: LISTEN, transport: clientTransport}

	segment, err := connection.transport.Receive()
	if err != nil {
		connection.transport.Close()
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Syn {
		connection.transport.Close()
		return nil, errors.New("did not receive syn for handshake")
	}

	connection.State = SYN_RECEIVED

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Syn: true, Ack: true}}, Message: &Message{MaxChars: 30, Protocol: "gbn"}})
	if err != nil {
		connection.transport.Close()
		return nil, err
	}

	segment, err = connection.transport.Receive()
	if err != nil {
		connection.transport.Close()
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Ack {
		connection.transport.Close()
		return nil, errors.New("did not receive ack for handshake")
	}

	connection.State = ESTABLISHED

	return connection, nil
}

func (l *Listener) Close() error {
	return l.transport.Close()
}
