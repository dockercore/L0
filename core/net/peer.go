package net

import (
	"net"
	"time"
)

type peer struct {
	lastActiveTime time.Time
	isRunning      bool
	conn           net.Conn
	peerID         []byte
}

func newPeer(c net.Conn) *peer {
	p := &peer{
		conn: c,
	}
	return p
}

func (p *peer) start() {
	// do something
}

func (p *peer) stop() {

}

func (p *peer) ping() {

}
