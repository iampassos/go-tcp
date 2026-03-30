package tcp

import (
	"net"
	"sync"
	"testing"
)

type transporterStub struct {
	send          func(segment Segment) error
	receive       func() (*Segment, error)
	receivedCount int
}

func (t *transporterStub) Send(segment Segment) error {
	if t.send == nil {
		return nil
	}
	return t.send(segment)
}

func (t *transporterStub) Receive() (*Segment, error) {
	return t.receive()
}

func (t *transporterStub) Close() error {
	return nil
}

func TestDial(t *testing.T) {
	l, err := net.Listen("tcp", ":46321")
	if err != nil {
		t.Fatalf("got an err on listen: %v", err)
	}
	defer l.Close()

	tests := []struct {
		name        string
		flags       Flags
		established bool
	}{
		{
			name:        "connection is established",
			flags:       Flags{Ack: true, Syn: true},
			established: true,
		},
		{
			name:  "connection isn't established",
			flags: Flags{Ack: false, Syn: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mu sync.Mutex
			defer mu.Unlock()
			mu.Lock()

			go func() {
				conn, err := l.Accept()
				if err != nil {
					t.Fatalf("got an err on accept: %v", err)
				}
				defer conn.Close()

				transport := Transport{conn: conn}
				err = transport.Send(Segment{Header: Header{Flags: tt.flags}})
				if err != nil {
					t.Fatalf("got an err on send: %v", err)
				}

				mu.Lock()
			}()

			conn, err := Dial(":46321")
			if !tt.established && err == nil {
				t.Fatalf("expected err, got nil")
			}

			if !tt.established {
				return
			}

			if tt.established && err != nil {
				t.Fatalf("got an err on dial: %v", err)
			}

			if tt.established && conn.State != ESTABLISHED {
				t.Fatalf("expected state %v, got %v", ESTABLISHED, conn.State)
			}

			err = conn.Transport.Close()
			if err != nil {
				t.Fatalf("got an err on close: %v", err)
			}
		})
	}

}

func TestListen(t *testing.T) {
	tests := []struct {
		name        string
		flags       []Flags
		established bool
	}{
		{
			name:        "connection is established",
			flags:       []Flags{Flags{Ack: false, Syn: true}, Flags{Ack: true, Syn: false}},
			established: true,
		},
		{
			name:        "connection isn't established with syn false",
			flags:       []Flags{Flags{Ack: false, Syn: false}, Flags{Ack: true, Syn: false}},
			established: false,
		},
		{
			name:        "connection isn't established with ack false",
			flags:       []Flags{Flags{Ack: false, Syn: true}, Flags{Ack: false, Syn: false}},
			established: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				conn, err := net.Dial("tcp", ":46321")
				if err != nil {
					t.Fatalf("got an err on dial: %v", err)
				}
				defer conn.Close()

				transport := Transport{conn: conn}
				for _, flags := range tt.flags {
					err = transport.Send(Segment{Header: Header{Flags: flags}})
					if err != nil {
						t.Fatalf("got an err on send: %v", err)
					}
					transport.Receive()
				}
			}()

			conn, err := Listen("46321")
			if !tt.established && err == nil {
				t.Fatalf("expected err, got nil")
			}

			if !tt.established {
				return
			}

			if tt.established && err != nil {
				t.Fatalf("got an err on listen: %v", err)
			}

			if tt.established && conn.State != ESTABLISHED {
				t.Fatalf("expected state %v, got %v", ESTABLISHED, conn.State)
			}

			err = conn.Transport.Close()
			if err != nil {
				t.Fatalf("got an err on close: %v", err)
			}
		})
	}
}
