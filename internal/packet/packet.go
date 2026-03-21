package packet

import (
	"github.com/iampassos/go-tcp/internal/tcp"
)

type Packet struct {
	Segment tcp.Segment
}
