package net

import (
	"crypto"
	"errors"

	"net"

	"sync"

	"github.com/bocheninc/L0/components/log"
)

// ServerConfig is the p2p network configuration
// Use default config when ServerConfig instance is nil
type ServerConfig struct {
	Address             string
	PrivateKey          *crypto.PrivateKey
	MaxPeers            int
	MinPeers            int
	ReconnectTimes      int
	ConnectTimeInterval int
	KeepAliveInterval   int
	KeepAliveTimes      int
	BootstrapNodes      []string
	RouteAddress        []string
	// Protocols           []Protocol
}

type server struct {
	ServerConfig
	isRunning     bool
	onConnectHook func(c net.Conn)
	listen        *net.TCPListener
	wg            sync.WaitGroup
}

// NewServer return p2p server instance
func NewServer(cfg ServerConfig) *server {
	s := &server{
		ServerConfig: cfg,
	}
	return s
}
func (s *server) Stop() (err error) {
	err = s.listen.Close()
	s.isRunning = false
	return
}

// Start the server when is stop
func (s *server) Start() error {
	if s.isRunning {
		return errors.New("net server already running")
	}

	log.Infoln("Trying start net server")

	if s.PrivateKey == nil {
		return errors.New("PrivateKey must be set to a non-nil key")
	}

	if s.Address != "" {
		s.wg.Add(1)
		go s.listening()
	}

	log.Infoln("net server start success")

	s.isRunning = true
	return nil
}

func (s *server) listening() (err error) {
	defer s.wg.Done()

	var (
		c    net.Conn
		addr *net.TCPAddr
	)

	addr, err = net.ResolveTCPAddr("tcp", s.Address)
	if err != nil {
		return
	}

	s.listen, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}

	for {
		c, err = s.listen.AcceptTCP()
		if err != nil {
			break
		}
		if s.onConnectHook != nil {
			s.onConnectHook(c)
		}
	}

	return
}

func (s *server) listenAddress() string {
	return s.listen.Addr().(*net.TCPAddr).String()
}

func (s *server) OnConnectHook(callback func(c net.Conn)) {
	s.onConnectHook = callback
}
