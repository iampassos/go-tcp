package tcp

type State int

const (
	CLOSED State = iota
	LISTEN
	SYN_SENT
	SYN_RECEIVED
	ESTABLISHED
)

type ClientTransporter interface {
	Send(segment Segment) error
	Receive() (*Segment, error)
	Close() error
	addr() string
}

type ServerTransporter interface {
	Accept() (ClientTransporter, error)
	Close() error
	addr() string
}

type Message struct {
	Protocol string
	MaxChars int
}

type Flags struct {
	Syn bool
	Ack bool
}

type Header struct {
	Flags      Flags
	WindowSize int
	Seq        int
	Ack        int
}

type Segment struct {
	Header  Header
	Message Message
}
