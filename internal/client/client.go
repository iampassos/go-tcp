package client

import (
	"github.com/iampassos/go-tcp/internal/packet"
	"github.com/iampassos/go-tcp/internal/tcp"
)

type State int

const (
	CLOSED State = iota
	SYN_SENT
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

func (c *Client) SendPacket(p packet.Packet) error {
	return nil
}
