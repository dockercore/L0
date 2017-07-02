package net

import (
	"errors"
	"net"
	"sync"

	"github.com/bocheninc/L0/components/log"
)

type server struct {
	isRunning      bool
	listenAddr     string
	connections    map[net.Conn]struct{}
	quit           chan struct{}
	addConn        chan string
	delConn        chan net.Conn
	onConnectHooks []func(c net.Conn)
	listener       *net.TCPListener
	wg             sync.WaitGroup
}

var s *server

// NewServer return p2p server instance
func newServer(listenAddress string) *server {
	if s == nil {
		s = &server{
			listenAddr:  listenAddress,
			connections: make(map[net.Conn]struct{}),
			quit:        make(chan struct{}),
			addConn:     make(chan string),
			delConn:     make(chan net.Conn),
		}
	}
	return s
}

// Stop the server
func (s *server) stop() (err error) {
	s.quit <- struct{}{}

	err = s.listener.Close()
	close(s.quit)

	for k := range s.connections {
		err = k.Close()
	}

	s.wg.Wait()
	s.isRunning = false

	return
}

// Start the server when is stop
func (s *server) start() error {
	if s.isRunning {
		return errors.New("net server already running")
	}

	log.Infoln("Trying start net server")

	if s.listenAddr != "" {
		s.wg.Add(1)
		go s.listen()
	}

	s.wg.Add(1)
	go s.run()

	log.Infoln("net server start success")

	s.isRunning = true
	s.wg.Wait()
	return nil
}

func (s *server) listen() (err error) {
	defer s.wg.Done()

	var (
		c    net.Conn
		addr *net.TCPAddr
	)

	addr, err = net.ResolveTCPAddr("tcp", s.listenAddr)
	if err != nil {
		return
	}

	s.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return
	}

	for {
		c, err = s.listener.AcceptTCP()
		if err != nil {
			break
		}

		s.connections[c] = struct{}{}

		for _, fn := range s.onConnectHooks {
			go fn(c)
		}
	}

	return
}

// ListenAddress return listen address
func (s *server) listenAddress() string {
	return s.listener.Addr().(*net.TCPAddr).String()
}

// OnConnectHook register callback function into server
// will be executed after accept new connection
func (s *server) onConnectHook(callback func(c net.Conn)) {
	s.onConnectHooks = append(s.onConnectHooks, callback)
}

// CloseEvent should be called after connection was closed
func (s *server) closeEvent(c net.Conn) {
	if _, ok := s.connections[c]; ok {
		delete(s.connections, c)
	}
}

func (s *server) run() {
	defer s.wg.Done()
	for {
		select {
		case <-s.quit:
			return
		case addr := <-s.addConn:
			s.wg.Add(1)
			go s.dial(addr)
		case c := <-s.delConn:
			c.Close()
			s.closeEvent(c)
		}
	}
}

// dial connects to a endpoint
func (s *server) dial(addr string) {
	defer s.wg.Done()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Error(err)
		return
	}

	s.connections[c] = struct{}{}

	for _, fn := range s.onConnectHooks {
		go fn(c)
	}
}
