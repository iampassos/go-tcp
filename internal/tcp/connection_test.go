package tcp

import (
	"sync"
	"testing"
)

func TestDial(t *testing.T) {
	tests := []struct {
		name    string
		segment Segment
		state   State
		wantErr bool
	}{
		{
			name:    "connection is established",
			segment: Segment{Header: Header{Flags: Flags{Syn: true, Ack: true}}},
			state:   ESTABLISHED,
		},
		{
			name:    "connection isnt established with syn false",
			segment: Segment{Header: Header{Flags: Flags{Syn: false, Ack: true}}},
			wantErr: true,
		},
		{
			name:    "connection isnt established with ack false",
			segment: Segment{Header: Header{Flags: Flags{Syn: true, Ack: false}}},
			wantErr: true,
		},
		{
			name:    "connection isnt established with syn/ack false",
			segment: Segment{Header: Header{Flags: Flags{Syn: false, Ack: false}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := InitServerTransport("0")
			if err != nil {
				t.Fatalf("error while listening: %v", err)
			}
			defer listener.Close()

			var mu sync.Mutex
			defer mu.Unlock()
			mu.Lock()

			go func() {
				conn, err := listener.Accept()
				if err != nil {
					listener.Close()
					return
				}
				defer conn.Close()
				conn.Send(tt.segment)
				mu.Lock()
			}()

			client, err := Dial(listener.listener.Addr().String(), Message{Protocol: "gbn", MaxChars: 30})
			if err != nil && !tt.wantErr {
				t.Fatalf("error while dialing: %v", err)
			}

			if err == nil && tt.wantErr {
				t.Fatalf("expected error, got %v", err)
			}

			if tt.wantErr {
				return
			}

			if client.State != tt.state {
				t.Fatalf("expected state %v, got %v", tt.state, client.State)
			}
		})
	}
}
