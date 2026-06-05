package tcp

import (
	"sync"
	"testing"
	"time"
)

type transportStub struct {
	in   chan Segment
	out  chan Segment
	peer string
}

func (t *transportStub) Send(segment Segment) error {
	t.out <- segment
	return nil
}

func (t *transportStub) Receive() (*Segment, error) {
	seg := <-t.in
	return &seg, nil
}

func (t *transportStub) Close() error {
	return nil
}

func (t *transportStub) addr() string {
	return t.peer
}

func NewStubPair() (*transportStub, *transportStub) {
	chanA := make(chan Segment, 1000)
	chanB := make(chan Segment, 1000)

	client := transportStub{in: chanB, out: chanA}
	server := transportStub{in: chanA, out: chanB}

	return &client, &server
}

func TestSend(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
		state    State
		faults   FaultConfig
		wantErr  error
		wantMsg  string
	}{
		{
			name:     "when connection is established sends message with gbn",
			protocol: GoBackN,
			state:    ESTABLISHED,
			wantMsg:  "Hello, World!",
		},
		{
			name:     "when connection is established sends message with sr",
			protocol: SelectiveRepeat,
			state:    ESTABLISHED,
			wantMsg:  "Hello, World!",
		},
		{
			name:     "when connection is established retransmits after dropped segment with gbn",
			protocol: GoBackN,
			state:    ESTABLISHED,
			faults:   FaultConfig{DropSegments: []int{2}},
			wantMsg:  "Hello, World!",
		},
		{
			name:     "when connection is established retransmits after corrupted segment with sr",
			protocol: SelectiveRepeat,
			state:    ESTABLISHED,
			faults:   FaultConfig{CorruptSegments: []int{2}},
			wantMsg:  "Hello, World!",
		},
		{
			name:     "when connection is not established errors",
			protocol: SelectiveRepeat,
			state:    CLOSED,
			wantErr:  ErrConnectionNotEstablished,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientTransport, serverTransport := NewStubPair()

			clientConn := &Connection{State: tt.state, transport: clientTransport, Protocol: tt.protocol, WindowSize: 5, Seq: 1, MaxChars: 30, Timeout: 20 * time.Millisecond, Faults: tt.faults}
			serverConn := &Connection{State: ESTABLISHED, transport: serverTransport, Protocol: tt.protocol, WindowSize: 5, Seq: 1, MaxChars: 30}

			done := make(chan string, 1)
			if tt.state == ESTABLISHED {
				go func() {
					msg, err := serverConn.Receive()
					if err != nil {
						t.Errorf("receive error: %v", err)
						return
					}
					done <- msg
				}()
			}

			err := clientConn.Send("Hello, World!")
			if tt.wantErr != err {
				t.Fatalf("expected error %v, got: %v", tt.wantErr, err)
			}

			if tt.wantErr != nil {
				return
			}

			err = clientConn.CloseWrite()
			if err != nil {
				t.Fatalf("error while closing write: %v", err)
			}

			select {
			case msg := <-done:
				if msg != tt.wantMsg {
					t.Fatalf(`expected message "%v", got: "%v"`, tt.wantMsg, msg)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for message")
			}
		})
	}
}

func TestReceive(t *testing.T) {
	tests := []struct {
		name     string
		protocol Protocol
		state    State
		wantErr  error
		wantMsg  string
		corrupt  bool
		wantNak  bool
	}{
		{
			name:     "when connection is established receives message with gbn",
			protocol: GoBackN,
			state:    ESTABLISHED,
			wantMsg:  "Hello, World!",
		},
		{
			name:     "when connection is established receives message with sr",
			protocol: SelectiveRepeat,
			state:    ESTABLISHED,
			wantMsg:  "Hello, World!",
		},
		{
			name:     "when connection is established rejects corrupted segment with nak",
			protocol: SelectiveRepeat,
			state:    ESTABLISHED,
			corrupt:  true,
			wantNak:  true,
		},
		{
			name:     "when connection is not established errors",
			protocol: SelectiveRepeat,
			state:    CLOSED,
			wantErr:  ErrConnectionNotEstablished,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientTransport, serverTransport := NewStubPair()

			clientConn := &Connection{State: ESTABLISHED, transport: clientTransport, Protocol: tt.protocol, WindowSize: 5, Seq: 1, MaxChars: 30}
			serverConn := &Connection{State: tt.state, transport: serverTransport, Protocol: tt.protocol, WindowSize: 5, Seq: 1, MaxChars: 30}

			var wg sync.WaitGroup

			if tt.corrupt {
				wg.Go(func() {
					good := newDataSegment(1, "Hell")
					corrupted := good
					corrupted.Message.Text = "Xell"
					serverTransport.in <- corrupted

					ack := <-serverTransport.out
					if ack.Header.Flags.Nak != tt.wantNak {
						t.Errorf("expected NAK %v, got flags %+v", tt.wantNak, ack.Header.Flags)
					}
					if ack.Header.Ack != 1 {
						t.Errorf("expected NAK 1, got %v", ack.Header.Ack)
					}

					serverTransport.in <- Segment{Header: Header{Flags: Flags{Fin: true}}}
				})
			} else {
				wg.Go(func() {
					clientConn.Send("Hello, World!")
					clientConn.CloseWrite()
				})
			}

			msg, err := serverConn.Receive()
			if tt.wantErr != err {
				t.Fatalf("expected error %v, got: %v", tt.wantErr, err)
			}

			if tt.wantMsg != msg {
				t.Fatalf(`expected message "%v", got: "%v"`, tt.wantMsg, msg)
			}

		})
	}
}

func newDataSegment(seq int, text string) Segment {
	segment := Segment{Header: Header{Seq: seq}, Message: Message{Text: text}}
	segment.Checksum = CalculateChecksum(segment)
	return segment
}

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

			client, err := Dial(listener.listener.Addr().String(), GoBackN, 30)
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
