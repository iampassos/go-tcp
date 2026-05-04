package tcp

import (
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

func (l *Listener) Accept(windowSize int) (*Connection, error) {
	if windowSize <= 0 || windowSize > 5 {
		return nil, ErrInvalidWindowSize
	}

	clientTransport, err := l.transport.Accept()
	if err != nil {
		return nil, err
	}

	connection := &Connection{State: LISTEN, transport: clientTransport, ISN: rand.Intn(1000), PeerAddr: clientTransport.addr()}

	segment, err := connection.transport.Receive()
	if err != nil {
		connection.transport.Close()
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Syn {
		connection.transport.Close()
		return nil, ErrSynNotReceived
	}

	connection.State = SYN_RECEIVED
	connection.MaxChars = segment.Message.MaxChars
	connection.Protocol = segment.Message.Protocol
	connection.WindowSize = windowSize
	connection.Seq = segment.Header.Seq + 1

	err = connection.transport.Send(Segment{
		Header:  Header{Flags: Flags{Syn: true, Ack: true}, WindowSize: connection.WindowSize, Ack: segment.Header.Seq + 1, Seq: connection.ISN},
		Message: Message{MaxChars: connection.MaxChars, Protocol: connection.Protocol}},
	)
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
		return nil, ErrAckNotReceived
	}

	connection.State = ESTABLISHED

	log.Printf("[SERVER] Connection established with %v. MaxChars: %v, Protocol: %v, WindowSize: %v", clientTransport.addr(), connection.MaxChars, connection.Protocol, connection.WindowSize)

	return connection, nil
}

func (l *Listener) Close() error {
	return l.transport.Close()
}
