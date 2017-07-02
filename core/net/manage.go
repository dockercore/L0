package net

import (
	"crypto"
	"golang.org/x/tools/go/gcimporter15/testdata"
	"net"
	"sync"
	"time"
)

// ServerConfig is the p2p network configuration
// Use default config when ServerConfig instance is nil
type PeerManagerCfg struct {
	Address             string
	PrivateKey          *crypto.PrivateKey
	MaxPeers            int
	MinPeers            int
	ReconnectTimes      int
	ConnectTimeInterval int64
	KeepAliveInterval   int64
	KeepAliveTimes      int
	BootstrapNodes      []string
	RouteAddress        []string
	// Protocols           []Protocol
}

type pm struct {
	PeerManagerCfg
	peers   map[net.Conn]*peer
	wg      sync.WaitGroup
	addPeer chan net.Conn
	delPeer chan *peer
	quit    chan struct{}
	srv     *server
}

var p *pm

func NewPM(cfg PeerManagerCfg) *pm {
	if p == nil {
		p = &pm{
			peers:   make(map[net.Conn]*peer, cfg.MaxPeers),
			addPeer: make(chan net.Conn),
			delPeer: make(chan *peer),
			quit:    make(chan struct{}),
		}
	}
	return p
}

func (p *pm) Start() {
	if p.Address != "" {
		p.srv = newServer(p.Address)
	}

	p.srv.onConnectHook(p.onConnectEvent)

	go p.srv.start()

	p.wg.Add(1)
	go p.run()

	p.wg.Wait()
}

func (p *pm) run() {
	defer p.wg.Done()
	ticker := time.NewTicker(time.Duration(p.KeepAliveInterval))
	for {
		select {
		case <-p.quit:
			return
		case c := <-p.addPeer:
			np := newPeer(c)
			p.peers[c] = np
			np.start()
		case dp := <-p.delPeer:
			dp.stop()
			delete(p.peers, dp.conn)
		case <-ticker.C:
			go p.check()
		}
	}
}

func (p *pm) onConnectEvent(c net.Conn) {

}

func (p *pm) check() {
	now := time.Now()
	for _, v := range p.peers {
		sec := int64(now.Sub(v.lastActiveTime))
		if sec > p.KeepAliveInterval*int64(p.KeepAliveTimes) {
			v.stop()
		}
		if sec > p.KeepAliveInterval {
			v.ping()
		}
	}
}

func (p *pm) Stop() {
	p.quit <- struct{}{}
	p.srv.stop()
	close(p.quit)
	close(p.addPeer)
	close(p.delPeer)
}
