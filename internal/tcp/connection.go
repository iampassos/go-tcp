package tcp

import (
	"log"
	"math/rand"
	"strings"
	"sync"
)

type Connection struct {
	ISN        int
	PeerAddr   string
	State      State
	transport  ClientTransporter
	Protocol   Protocol
	MaxChars   int
	WindowSize int
	Seq        int
}

func Dial(addr string, protocol Protocol, maxChars int) (*Connection, error) {
	if maxChars < 30 {
		return nil, ErrMaxCharsMinimum
	}

	if protocol != SelectiveRepeat && protocol != GoBackN {
		return nil, ErrInvalidProtocol
	}

	clientTransport, err := InitClientTransport(addr)
	if err != nil {
		return nil, err
	}

	connection := &Connection{State: CLOSED, transport: clientTransport, ISN: rand.Intn(1000), PeerAddr: addr}

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Syn: true}, Seq: connection.ISN}, Message: Message{MaxChars: maxChars, Protocol: protocol}})
	if err != nil {
		return nil, err
	}

	connection.State = SYN_SENT

	segment, err := connection.transport.Receive()
	if err != nil {
		return nil, err
	}

	if segment == nil || !segment.Header.Flags.Syn || !segment.Header.Flags.Ack {
		return nil, ErrSynAckNotReceived
	}

	connection.State = ESTABLISHED
	connection.Protocol = segment.Message.Protocol
	connection.MaxChars = segment.Message.MaxChars
	connection.WindowSize = segment.Header.WindowSize
	connection.Seq = connection.ISN + 1

	err = connection.transport.Send(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: segment.Header.Seq + 1, Seq: segment.Header.Ack}})
	if err != nil {
		return nil, err
	}

	log.Printf("[CLIENT] Connection established with %v. MaxChars: %v, Protocol: %v, WindowSize: %v", clientTransport.conn.RemoteAddr().String(), connection.MaxChars, connection.Protocol, connection.WindowSize)

	return connection, nil
}

func (c *Connection) Receive() (string, error) {
	if c.State != ESTABLISHED {
		return "", ErrConnectionNotEstablished
	}

	var buffer []string
	expectedSeq := c.Seq

	for {
		segment, err := c.transport.Receive()
		if err != nil {
			return "", err
		}

		if segment.Header.Flags.Fin {
			log.Printf(`[SERVER] Received segment with FIN flag from %v`, c.PeerAddr)
			break
		}

		log.Printf(`[SERVER] Received segment with text "%v" and SEQ %v from %v`, segment.Message.Text, segment.Header.Seq, c.PeerAddr)

		seqNum := segment.Header.Seq

		if seqNum == expectedSeq {
			buffer = append(buffer, segment.Message.Text)

			err := c.transport.Send(Segment{Header{Ack: seqNum}, Message{}})
			if err != nil {
				return "", err
			}

			log.Printf(`[SERVER] Sending segment with ACK %v to %v`, seqNum, c.PeerAddr)

			expectedSeq = seqNum + 1

		} else {
			err := c.transport.Send(Segment{Header{Ack: expectedSeq - 1}, Message{}})
			if err != nil {
				return "", err
			}
		}

	}

	text := strings.Join(buffer, "")

	log.Printf(`[SERVER] Received message "%s" from %v`, text, c.PeerAddr)

	c.Seq = expectedSeq

	return text, nil
}

func (c *Connection) Send(text string) error {
	if c.State != ESTABLISHED {
		return ErrConnectionNotEstablished
	}

	if len(text) > c.MaxChars {
		return ErrMaxCharsExceeded
	}

	log.Printf(`[CLIENT] Sending message "%s"`, text)

	var window []Segment
	maxChars := 4

	seq := c.Seq
	for i := 0; i < len(text); i += maxChars {
		end := min(i+maxChars, len(text))
		t := text[i:end]
		window = append(window, Segment{Header{Seq: seq}, Message{Text: t}})
		seq++
	}

	var wg sync.WaitGroup
	ch := make(chan int, len(window))
	base := 0
	nextSeq := 0

	wg.Go(func() {
		for nextSeq < len(window) {
			if nextSeq < base+c.WindowSize {
				segment := window[nextSeq]
				err := c.transport.Send(segment)
				if err != nil {
					return
				}

				log.Printf(`[CLIENT] Sending segment with text "%v" and SEQ %v`, segment.Message.Text, segment.Header.Seq)
				nextSeq++
			} else {
				<-ch
			}
		}
	})

	wg.Go(func() {
		for base < len(window) {
			segment, err := c.transport.Receive()
			if err != nil {
				return
			}

			ackNum := segment.Header.Ack
			ackIndex := ackNum - c.Seq

			log.Printf(`[CLIENT] Received segment with ACK %v`, segment.Header.Ack)

			if c.Protocol == GoBackN {
				if ackIndex >= base {
					base = ackIndex + 1
					ch <- base
				}
			}
		}
	})

	wg.Wait()

	c.Seq += len(window)

	return nil
}

func (c *Connection) CloseWrite() error {
	err := c.transport.Send(Segment{Header: Header{Flags: Flags{Fin: true}}})
	if err != nil {
		return err
	}

	return nil
}

func (c *Connection) Close() error {
	log.Printf("[HOST] Connection closed with %v", c.PeerAddr)

	return c.transport.Close()
}
