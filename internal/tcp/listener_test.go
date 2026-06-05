package tcp

import (
	"testing"
)

func TestListener(t *testing.T) {
	tests := []struct {
		name     string
		segments []Segment
		key      string
		state    State
		wantErr  bool
	}{
		{
			name: "connection is established",
			segments: []Segment{
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: true, Ack: false}}, Message: Message{Protocol: GoBackN, MaxChars: 30}}),
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: false, Ack: true}}}),
			},
			state: ESTABLISHED,
		},
		{
			name: "encrypted connection is established",
			segments: []Segment{
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: true, Ack: false}}, Message: Message{Protocol: GoBackN, MaxChars: 30, Encrypted: true, KeyHash: keyHash("secret")}}),
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: false, Ack: true}}}),
			},
			key:   "secret",
			state: ESTABLISHED,
		},
		{
			name: "connection isnt established with first syn false",
			segments: []Segment{
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: false, Ack: false}}}),
			},
			wantErr: true,
		},
		{
			name: "connection isnt established with second ack false",
			segments: []Segment{
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: true, Ack: false}}, Message: Message{Protocol: GoBackN, MaxChars: 30}}),
				withChecksum(Segment{Header: Header{Flags: Flags{Syn: false, Ack: false}}}),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := Listen("0")
			if err != nil {
				t.Fatalf("error while listening: %v", err)
			}
			defer listener.Close()

			go func() {
				client, err := InitClientTransport(listener.transport.addr())
				if err != nil {
					return
				}
				defer client.Close()

				for i, s := range tt.segments {
					client.Send(s)
					if i == 0 && s.Header.Flags.Syn {
						client.Receive()
					}
				}
			}()

			connection, err := listener.AcceptWithKey(5, tt.key)

			if err != nil && !tt.wantErr {
				t.Fatalf("error while accepting: %v", err)
			}

			if err == nil && tt.wantErr {
				t.Fatalf("expected error, got %v", err)
			}

			if tt.wantErr {
				return
			}

			if connection.State != tt.state {
				t.Fatalf("expected state %v, got %v", tt.state, connection.State)
			}
		})
	}
}
