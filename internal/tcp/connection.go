package tcp

import (
	"hash/crc32"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
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
	Timeout    time.Duration
	Faults     FaultConfig
	ackOnce    sync.Once
	ackCh      chan Segment
	ackErrCh   chan error
}

const defaultTimeout = time.Second

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

	buffer := make(map[int]string)
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
		if !ValidChecksum(*segment) {
			err := c.transport.Send(Segment{Header: Header{Flags: Flags{Nak: true}, Ack: seqNum}})
			if err != nil {
				return "", err
			}

			log.Printf(`[SERVER] Sending segment with NAK %v to %v`, seqNum, c.PeerAddr)
			continue
		}

		if c.Protocol == GoBackN {
			if seqNum == expectedSeq {
				buffer[seqNum] = segment.Message.Text

				err := c.transport.Send(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: seqNum}})
				if err != nil {
					return "", err
				}

				log.Printf(`[SERVER] Sending segment with ACK %v to %v`, seqNum, c.PeerAddr)

				expectedSeq = seqNum + 1
			} else {
				err := c.transport.Send(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: expectedSeq - 1}})
				if err != nil {
					return "", err
				}
			}
		}

		if c.Protocol == SelectiveRepeat {
			err := c.transport.Send(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: seqNum}})
			if err != nil {
				return "", err
			}

			log.Printf(`[SERVER] Sending segment with ACK %v to %v`, seqNum, c.PeerAddr)

			buffer[seqNum] = segment.Message.Text
		}

	}

	keys := make([]int, 0, len(buffer))
	for k := range buffer {
		keys = append(keys, k)
	}

	sort.Ints(keys)

	values := make([]string, 0, len(buffer))
	for _, k := range keys {
		values = append(values, buffer[k])
	}

	text := strings.Join(values, "")

	log.Printf(`[SERVER] Received message "%s" from %v`, text, c.PeerAddr)

	if len(keys) > 0 {
		c.Seq = keys[len(keys)-1] + 1
	}

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
		segment := Segment{Header: Header{Seq: seq}, Message: Message{Text: t}}
		segment.Checksum = CalculateChecksum(segment)
		window = append(window, segment)
		seq++
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	base := 0
	nextSeq := 0
	acked := make([]bool, len(window))
	c.ensureAckReader(len(window) * 4)

	dropped := setFromInts(c.Faults.DropSegments)
	corrupted := setFromInts(c.Faults.CorruptSegments)

	sendSegment := func(index int) error {
		segment := window[index]
		segmentNumber := index + 1
		seqNum := segment.Header.Seq

		if dropped[segmentNumber] {
			delete(dropped, segmentNumber)
			log.Printf(`[CLIENT] Simulating loss of message segment %v with text "%v" and SEQ %v`, segmentNumber, segment.Message.Text, seqNum)
			return nil
		}

		if corrupted[segmentNumber] {
			delete(corrupted, segmentNumber)
			segment.Message.Text = corruptText(segment.Message.Text)
			log.Printf(`[CLIENT] Simulating corruption of message segment %v with SEQ %v`, segmentNumber, seqNum)
		}

		err := c.transport.Send(segment)
		if err != nil {
			return err
		}

		log.Printf(`[CLIENT] Sending segment with text "%v" and SEQ %v`, segment.Message.Text, seqNum)
		return nil
	}

	for base < len(window) {
		for nextSeq < len(window) && nextSeq < base+c.WindowSize {
			if err := sendSegment(nextSeq); err != nil {
				return err
			}
			nextSeq++
		}

		select {
		case segment := <-c.ackCh:
			ackNum := segment.Header.Ack
			ackIndex := ackNum - c.Seq

			if ackIndex < 0 || ackIndex >= len(window) {
				continue
			}

			if segment.Header.Flags.Nak {
				log.Printf(`[CLIENT] Received segment with NAK %v`, ackNum)
				if c.Protocol == GoBackN {
					nextSeq = ackIndex
					continue
				}
				if err := sendSegment(ackIndex); err != nil {
					return err
				}
				continue
			}

			log.Printf(`[CLIENT] Received segment with ACK %v`, ackNum)

			if c.Protocol == GoBackN && ackIndex >= base {
				for i := base; i <= ackIndex; i++ {
					acked[i] = true
				}
				base = ackIndex + 1
			}
			if c.Protocol == SelectiveRepeat {
				acked[ackIndex] = true
				for base < len(window) && acked[base] {
					base++
				}
			}
		case err := <-c.ackErrCh:
			return err
		case <-time.After(timeout):
			log.Printf(`[CLIENT] Timeout waiting for ACK at SEQ %v`, window[base].Header.Seq)
			if c.Protocol == GoBackN {
				nextSeq = base
				continue
			}
			for i := base; i < nextSeq; i++ {
				if !acked[i] {
					if err := sendSegment(i); err != nil {
						return err
					}
				}
			}
		}
	}

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

func (c *Connection) ensureAckReader(bufferSize int) {
	c.ackOnce.Do(func() {
		c.ackCh = make(chan Segment, bufferSize)
		c.ackErrCh = make(chan error, 1)
		go func() {
			for {
				segment, err := c.transport.Receive()
				if err != nil {
					c.ackErrCh <- err
					return
				}
				c.ackCh <- *segment
			}
		}()
	})
}

func CalculateChecksum(segment Segment) uint32 {
	data := []byte(segment.Message.Text)
	data = append(data, []byte(segment.Message.Protocol)...)
	data = append(data, byte(segment.Message.MaxChars))
	data = append(data, byte(segment.Header.Seq), byte(segment.Header.Ack), byte(segment.Header.WindowSize))
	return crc32.ChecksumIEEE(data)
}

func ValidChecksum(segment Segment) bool {
	return segment.Checksum == CalculateChecksum(segment)
}

func setFromInts(values []int) map[int]bool {
	set := make(map[int]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func corruptText(text string) string {
	if text == "" {
		return "!"
	}
	return "!" + text[1:]
}
