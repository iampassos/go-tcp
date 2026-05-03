package tcp

import (
	"errors"
	"log"
	"math/rand"
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

	connection := &Connection{State: LISTEN, transport: clientTransport, ISN: rand.Int()}

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
	connection.MaxChars = segment.Message.MaxChars
	connection.Protocol = segment.Message.Protocol
	connection.WindowSize = 5

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Syn: true, Ack: true}, WindowSize: connection.WindowSize, Ack: segment.Header.Seq + 1, Seq: connection.ISN}, Message: Message{MaxChars: connection.MaxChars, Protocol: connection.Protocol}})
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

	log.Printf("[SERVER] Connection established with %v. MaxChars: %v, Protocol: %v, WindowSize: %v", clientTransport.addr(), connection.MaxChars, connection.Protocol, connection.WindowSize)

	return connection, nil
}

func (l *Listener) Close() error {
	return l.transport.Close()
}
