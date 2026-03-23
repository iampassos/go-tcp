package tcp

import (
	"testing"
)

type NetworkLayerStub struct {
	sendSegment    func(Segment) error
	receiveSegment func() (*Segment, error)
}

func (nl *NetworkLayerStub) SendSegment(s Segment) error {
	return nl.sendSegment(s)
}

func (nl *NetworkLayerStub) ReceiveSegment() (*Segment, error) {
	return nl.receiveSegment()
}

func TestStartHandshake(t *testing.T) {
	stub := NetworkLayerStub{sendSegment: func(s Segment) error { return nil }}
	connection := NewConnection(&stub)

	err := connection.startHandshake()
	if err != nil {
		t.Fatal(err)
	}

	if connection.State != SYN_SENT {
		t.Fatalf("got state %d, wanted %d", connection.State, SYN_SENT)
	}

}

func TestReceiveHandshake(t *testing.T) {
	stub := NetworkLayerStub{
		receiveSegment: func() (*Segment, error) {
			return &Segment{Header: Header{Syn: true, Ack: true}}, nil
		}}
	connection := NewConnection(&stub)

	err := connection.receiveHandshake()
	if err != nil {
		t.Fatal(err)
	}

	if connection.State != SYN_RECEIVED {
		t.Fatalf("got state %d, wanted %d", connection.State, SYN_RECEIVED)
	}
}

func TestEndHandshake(t *testing.T) {
	stub := NetworkLayerStub{sendSegment: func(s Segment) error { return nil }}
	connection := NewConnection(&stub)

	err := connection.endHandshake()
	if err != nil {
		t.Fatal(err)
	}

	if connection.State != ESTABLISHED {
		t.Fatalf("got state %d, wanted %d", connection.State, SYN_RECEIVED)
	}
}
