package client

import (
	"testing"

	"github.com/iampassos/go-tcp/internal/packet"
	"github.com/iampassos/go-tcp/internal/tcp"
)

type NetworkLayerStub struct {
	sendPacket    func(packet.Packet) error
	receivePacket func() (*packet.Packet, error)
}

func (nl *NetworkLayerStub) SendPacket(p packet.Packet) error {
	return nl.sendPacket(p)
}

func (nl *NetworkLayerStub) ReceivePacket() (*packet.Packet, error) {
	return nl.receivePacket()
}

func TestHandshake(t *testing.T) {
	stub := NetworkLayerStub{
		sendPacket: func(p packet.Packet) error {
			return nil
		},
	}
	client := NewClient(&stub)

	t.Run("client sends SYN and changes state", func(t *testing.T) {
		err := client.StartHandshake()
		if err != nil {
			t.Fatal(err)
		}

		if client.State != SYN_SENT {
			t.Fatalf("got state %d, wanted %d", client.State, SYN_SENT)
		}
	})

	stub = NetworkLayerStub{
		sendPacket: stub.sendPacket,
		receivePacket: func() (*packet.Packet, error) {
			return &packet.Packet{Segment: tcp.Segment{Header: tcp.Header{Syn: true, Ack: true}}}, nil
		},
	}

	t.Run("client receives SYN/ACK and changes state", func(t *testing.T) {
		err := client.ReceiveHandshake()
		if err != nil {
			t.Fatal(err)
		}

		if client.State != SYN_RECEIVED {
			t.Fatalf("got state %d, wanted %d", client.State, SYN_RECEIVED)
		}
	})
}
