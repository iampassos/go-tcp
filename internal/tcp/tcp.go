package tcp

import (
	"errors"
)

type State int

const (
	CLOSED State = iota
	LISTEN
	SYN_SENT
	SYN_RECEIVED
	ESTABLISHED
)

type Transporter interface {
	Send(segment Segment) error
	Receive() (*Segment, error)
	Close() error
}

type Flags struct {
	Syn bool
	Ack bool
}

type Header struct {
	Flags Flags
}

type Segment struct {
	Header Header
}

type Conn struct {
	State     State
	Transport Transporter
}

func Dial(addr string) (*Conn, error) {
	transporter, err := InitClientTransport(addr)
	if err != nil {
		return nil, err
	}

	conn := &Conn{State: CLOSED}

	err = transporter.Send(Segment{Header: Header{Flags: Flags{Syn: true}}})
	if err != nil {
		return nil, err
	}

	conn.State = SYN_SENT

	segment, err := transporter.Receive()
	if err != nil {
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Syn || !segment.Header.Flags.Ack {
		return nil, errors.New("syn/ack flags not received")
	}

	conn.State = ESTABLISHED

	err = transporter.Send(Segment{Header: Header{Flags: Flags{Ack: true}}})
	if err != nil {
		return nil, err
	}

	conn.Transport = transporter

	return conn, nil
}

func Listen(port string) (*Conn, error) {
	transporter, err := InitServerTransport(port)
	if err != nil {
		return nil, err
	}

	conn := &Conn{State: LISTEN}

	segment, err := transporter.Receive()
	if err != nil {
		transporter.Close()
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Syn {
		transporter.Close()
		return nil, errors.New("did not receive syn for handshake")
	}

	conn.State = SYN_RECEIVED

	err = transporter.Send(Segment{Header: Header{Flags: Flags{Syn: true, Ack: true}}})
	if err != nil {
		transporter.Close()
		return nil, err
	}

	segment, err = transporter.Receive()
	if err != nil {
		transporter.Close()
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Ack {
		transporter.Close()
		return nil, errors.New("did not receive ack for handshake")
	}

	conn.State = ESTABLISHED

	conn.Transport = transporter

	return conn, nil
}
