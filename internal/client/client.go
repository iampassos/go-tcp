package client

import (
	"errors"

	"github.com/iampassos/go-tcp/internal/packet"
	"github.com/iampassos/go-tcp/internal/tcp"
)

type State int

const (
	CLOSED State = iota
	SYN_SENT
	SYN_RECEIVED
)

var (
	ErrSynAckNotReceived = errors.New("SYN/ACK was not received")
)

type Client struct {
	State State
}

func NewClient() *Client {
	return &Client{State: CLOSED}
}

func (c *Client) StartHandshake() error {
	packet := packet.Packet{Segment: tcp.Segment{Header: tcp.Header{Syn: true}}}

	err := c.SendPacket(packet)
	if err != nil {
		return err
	}

	c.State = SYN_SENT

	return nil
}

func (c *Client) ReceiveHandshake() error {
	packet, err := c.ReceivePacket()
	if err != nil {
		return err
	}

	if packet.Segment.Header.Syn && packet.Segment.Header.Ack {
		c.State = SYN_RECEIVED
		return nil
	}

	return ErrSynAckNotReceived
}

func (c *Client) ReceivePacket() (*packet.Packet, error) {
	packet := packet.Packet{Segment: tcp.Segment{Header: tcp.Header{Syn: true, Ack: true}}}
	return &packet, nil
}

func (c *Client) SendPacket(p packet.Packet) error {
	return nil
}
