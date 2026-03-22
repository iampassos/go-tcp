package tcp

import (
	"errors"
)

type State int

const (
	CLOSED State = iota
	SYN_SENT
	SYN_RECEIVED
	ESTABLISHED
)

var (
	ErrSynAckNotReceived = errors.New("SYN/ACK was not received")
)

type Header struct {
	Syn bool
	Ack bool
}

type Segment struct {
	Header Header
}

type NetworkLayer interface {
	SendSegment(Segment) error
	ReceiveSegment() (*Segment, error)
}

type Connection struct {
	Network NetworkLayer
	State   State
}

func NewConnection(n NetworkLayer) *Connection {
	return &Connection{Network: n, State: CLOSED}
}

func (c *Connection) startHandshake() error {
	segment := Segment{Header{Syn: true}}

	err := c.Network.SendSegment(segment)
	if err != nil {
		return err
	}

	c.State = SYN_SENT

	return nil
}

func (c *Connection) receiveHandshake() error {
	segment, err := c.Network.ReceiveSegment()
	if err != nil {
		return err
	}

	if segment.Header.Syn && segment.Header.Ack {
		c.State = SYN_RECEIVED
		return nil
	}

	return ErrSynAckNotReceived
}

func (c *Connection) endHandshake() error {
	segment := Segment{Header: Header{Ack: true}}

	err := c.Network.SendSegment(segment)
	if err != nil {
		return err
	}

	c.State = ESTABLISHED

	return nil
}
