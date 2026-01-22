package outbound

import (
	"errors"
	"net"
)

var errRejected = errors.New("connection rejected")

// Reject is an Outbound that rejects all connections.
type Reject struct{}

// NewReject creates a new Reject outbound.
func NewReject() Outbound {
	return &Reject{}
}

// DialTCP always returns an error.
func (r *Reject) DialTCP(addr *Addr) (net.Conn, error) {
	return nil, errRejected
}

// DialUDP always returns an error.
func (r *Reject) DialUDP(addr *Addr) (UDPConn, error) {
	return nil, errRejected
}
