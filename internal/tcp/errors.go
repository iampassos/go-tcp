package tcp

import "errors"

var (
	ErrConnectionNotEstablished = errors.New("connection isn't established")
	ErrMaxCharsExceeded         = errors.New("max chars exceeded for this text")
	ErrMaxCharsMinimum          = errors.New("max chars must be at least 30")
	ErrInvalidProtocol          = errors.New("protocol must be either gbn or sr")
	ErrSynAckNotReceived        = errors.New("syn/ack flags not received")
	ErrSynNotReceived           = errors.New("syn flag not received")
	ErrAckNotReceived           = errors.New("ack flag not received")
	ErrInvalidWindowSize        = errors.New("window size must be between 1 and 5")
)

type Protocol string

const (
	GoBackN         = "gbn"
	SelectiveRepeat = "sr"
)
