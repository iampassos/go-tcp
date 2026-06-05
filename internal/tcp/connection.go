package tcp

import (
	"encoding/binary"
	"log"
	"math/rand"
	"sort"
	"strconv"
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

	err = connection.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Syn: true}, Seq: connection.ISN}, Message: Message{MaxChars: maxChars, Protocol: protocol}}))
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
	if !ValidChecksum(*segment) {
		return nil, ErrSynAckNotReceived
	}

	connection.State = ESTABLISHED
	connection.Protocol = segment.Message.Protocol
	connection.MaxChars = segment.Message.MaxChars
	connection.WindowSize = segment.Header.WindowSize
	connection.Seq = connection.ISN + 1

	err = connection.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: segment.Header.Seq + 1, Seq: segment.Header.Ack}}))
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

		log.Printf(`[SERVER] Received segment metadata: %s from %v`, formatSegmentMetadata(*segment), c.PeerAddr)

		seqNum := segment.Header.Seq
		if !ValidChecksum(*segment) {
			err := c.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Nak: true}, Ack: seqNum}}))
			if err != nil {
				return "", err
			}

			log.Printf(`[SERVER] Sending confirmation flags=%+v ack=%v to %v`, Flags{Nak: true}, seqNum, c.PeerAddr)
			continue
		}

		if segment.Header.Flags.Fin {
			log.Printf(`[SERVER] Received segment with FIN flag from %v`, c.PeerAddr)
			break
		}

		if c.Protocol == GoBackN {
			if seqNum == expectedSeq {
				buffer[seqNum] = segment.Message.Text

				err := c.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: seqNum}}))
				if err != nil {
					return "", err
				}

				log.Printf(`[SERVER] Sending confirmation flags=%+v ack=%v to %v`, Flags{Ack: true}, seqNum, c.PeerAddr)

				expectedSeq = seqNum + 1
			} else {
				err := c.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: expectedSeq - 1}}))
				if err != nil {
					return "", err
				}
				log.Printf(`[SERVER] Sending confirmation flags=%+v ack=%v to %v`, Flags{Ack: true}, expectedSeq-1, c.PeerAddr)
			}
		}

		if c.Protocol == SelectiveRepeat {
			err := c.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Ack: true}, Ack: seqNum}}))
			if err != nil {
				return "", err
			}

			log.Printf(`[SERVER] Sending confirmation flags=%+v ack=%v to %v`, Flags{Ack: true}, seqNum, c.PeerAddr)

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

	runes := []rune(text)
	if len(runes) > c.MaxChars {
		return ErrMaxCharsExceeded
	}

	log.Printf(`[CLIENT] Sending message "%s"`, text)

	var window []Segment
	maxChars := 4

	seq := c.Seq
	for i := 0; i < len(runes); i += maxChars {
		end := min(i+maxChars, len(runes))
		t := string(runes[i:end])
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
			if !ValidChecksum(segment) {
				log.Printf(`[CLIENT] Ignoring invalid confirmation metadata: %s`, formatSegmentMetadata(segment))
				continue
			}

			ackNum := segment.Header.Ack
			ackIndex := ackNum - c.Seq

			if ackIndex < 0 || ackIndex >= len(window) {
				continue
			}

			if segment.Header.Flags.Nak {
				log.Printf(`[CLIENT] Received confirmation metadata: %s`, formatSegmentMetadata(segment))
				if c.Protocol == GoBackN {
					nextSeq = ackIndex
					continue
				}
				if err := sendSegment(ackIndex); err != nil {
					return err
				}
				continue
			}

			log.Printf(`[CLIENT] Received confirmation metadata: %s`, formatSegmentMetadata(segment))

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
	err := c.transport.Send(withChecksum(Segment{Header: Header{Flags: Flags{Fin: true}}}))
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
	return uint32(oneComplementChecksum(checksumBytes(segment)))
}

func withChecksum(segment Segment) Segment {
	segment.Checksum = CalculateChecksum(segment)
	return segment
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
	runes := []rune(text)
	runes[0] = '!'
	return string(runes)
}

func formatSegmentMetadata(segment Segment) string {
	return "flags=" + formatFlags(segment.Header.Flags) +
		" seq=" + strconv.Itoa(segment.Header.Seq) +
		" ack=" + strconv.Itoa(segment.Header.Ack) +
		" window=" + strconv.Itoa(segment.Header.WindowSize) +
		" checksum=" + strconv.FormatUint(uint64(segment.Checksum), 10) +
		" calculatedChecksum=" + strconv.FormatUint(uint64(CalculateChecksum(segment)), 10) +
		` text="` + segment.Message.Text + `"` +
		" protocol=" + string(segment.Message.Protocol) +
		" maxChars=" + strconv.Itoa(segment.Message.MaxChars)
}

func formatFlags(flags Flags) string {
	values := make([]string, 0, 4)
	if flags.Syn {
		values = append(values, "SYN")
	}
	if flags.Ack {
		values = append(values, "ACK")
	}
	if flags.Nak {
		values = append(values, "NAK")
	}
	if flags.Fin {
		values = append(values, "FIN")
	}
	if len(values) == 0 {
		return "NONE"
	}
	return strings.Join(values, "|")
}

func checksumBytes(segment Segment) []byte {
	var data []byte
	data = appendStringForChecksum(data, segment.Message.Text)
	data = appendStringForChecksum(data, string(segment.Message.Protocol))
	data = appendIntForChecksum(data, segment.Message.MaxChars)
	data = appendIntForChecksum(data, segment.Header.Seq)
	data = appendIntForChecksum(data, segment.Header.Ack)
	data = appendIntForChecksum(data, segment.Header.WindowSize)
	data = appendBoolForChecksum(data, segment.Header.Flags.Syn)
	data = appendBoolForChecksum(data, segment.Header.Flags.Ack)
	data = appendBoolForChecksum(data, segment.Header.Flags.Nak)
	data = appendBoolForChecksum(data, segment.Header.Flags.Fin)
	return data
}

func appendStringForChecksum(data []byte, value string) []byte {
	data = appendIntForChecksum(data, len(value))
	return append(data, []byte(value)...)
}

func appendIntForChecksum(data []byte, value int) []byte {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], uint64(value))
	return append(data, encoded[:]...)
}

func appendBoolForChecksum(data []byte, value bool) []byte {
	if value {
		return append(data, 1)
	}
	return append(data, 0)
}

func oneComplementChecksum(data []byte) uint16 {
	var sum uint32
	for i := 0; i < len(data); i += 2 {
		word := uint16(data[i]) << 8
		if i+1 < len(data) {
			word |= uint16(data[i+1])
		}
		sum += uint32(word)
		sum = (sum & 0xffff) + (sum >> 16)
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}
