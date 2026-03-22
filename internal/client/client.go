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

type NetworkLayer interface {
	SendPacket(packet.Packet) error
	ReceivePacket() (*packet.Packet, error)
}

type Client struct {
	NetworkLayer NetworkLayer
	State        State
}

func NewClient(nl NetworkLayer) *Client {
	return &Client{NetworkLayer: nl, State: CLOSED}
}

func (c *Client) StartHandshake() error {
	packet := packet.Packet{Segment: tcp.Segment{Header: tcp.Header{Syn: true}}}

	err := c.NetworkLayer.SendPacket(packet)
	if err != nil {
		return err
	}

	c.State = SYN_SENT

	return nil
}

func (c *Client) ReceiveHandshake() error {
	packet, err := c.NetworkLayer.ReceivePacket()
	if err != nil {
		return err
	}

	if packet.Segment.Header.Syn && packet.Segment.Header.Ack {
		c.State = SYN_RECEIVED
		return nil
	}

	return ErrSynAckNotReceived
}
