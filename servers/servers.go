// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package servers

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// Logger defines methods required for logging.
type Logger interface {
	Infof(format string, a ...any)
	Errorf(format string, a ...any)
}

// stdLogger is a simple implementation of Logger interface
// that uses log package for logging messages.
type stdLogger struct{}

func (l stdLogger) Infof(format string, a ...any) {
	log.Printf("INFO "+format, a...)
}

func (l stdLogger) Errorf(format string, a ...any) {
	log.Printf("ERROR "+format, a...)
}

// Option is a function that sets optional parameters for Servers.
type Option func(*Servers)

// WithLogger sets the Logger instance for logging messages.
func WithLogger(logger Logger) Option { return func(o *Servers) { o.logger = logger } }

// WithRecoverFunc sets a function that will be used to recover
// from panic inside a goroutune that servers are serving requests.
func WithRecoverFunc(recover func()) Option { return func(o *Servers) { o.recover = recover } }

// Servers holds a list of servers and their options.
// It provides a simple way to construct server group with Add method,
// to start them with Serve method, and stop them with Close or Shutdown methods.
type Servers struct {
	servers []*server
	mu      sync.Mutex
	logger  Logger
	recover func()
}

// New creates a new instance of Servers with applied options.
func New(opts ...Option) (s *Servers) {
	s = &Servers{
		logger:  stdLogger{},
		recover: func() {},
	}
	for _, opt := range opts {
		opt(s)
	}
	return
}

// Server defines required methods for a type that can be added to
// the Servers.
// In addition to this methods, a Server should implement TCPServer
// or UDPServer to be able to serve requests.
type Server interface {
	// Close should stop server from serving all existing requests
	// and stop accepting new ones.
	// The listener provided in Serve method must stop listening.
	Close() error
	// Shutdown should gracefully stop server. All existing requests
	// should be processed within a deadline provided by the context.
	// No new requests should be accepted.
	// The listener provided in Serve method must stop listening.
	Shutdown(ctx context.Context) error
}

// TCPServer defines methods for a server that accepts requests
// over TCP listener.
type TCPServer interface {
	// Serve should start server responding to requests.
	// The listener is initialized and already listening.
	ServeTCP(ln net.Listener) error
}

// UDPServer defines methods for a server that accepts requests
// over UDP listener.
type UDPServer interface {
	ServeUDP(conn *net.UDPConn) error
}

type server struct {
	Server
	name    string
	address string
	tcpAddr *net.TCPAddr
	udpAddr *net.UDPAddr
}

func (s *server) label() string {
	if s.name == "" {
		return "server"
	}
	return s.name + " server"
}

func (s *server) isTCP() (srv TCPServer, yes bool) {
	srv, yes = s.Server.(TCPServer)
	return
}

func (s *server) isUDP() (srv UDPServer, yes bool) {
	srv, yes = s.Server.(UDPServer)
	return
}

// Add adds a new server instance by a custom name and with
// address to listen to.
func (s *Servers) Add(name, address string, srv Server) {
	s.mu.Lock()
	s.servers = append(s.servers, &server{
		Server:  srv,
		name:    name,
		address: address,
	})
	s.mu.Unlock()
}

// Serve starts all added servers.
// New new servers must be added after this methid is called.
func (s *Servers) Serve() (err error) {
	lns := make([]net.Listener, len(s.servers))
	conns := make([]*net.UDPConn, len(s.servers))
	for i, srv := range s.servers {
		if _, yes := srv.isTCP(); yes {
			ln, err := net.Listen("tcp", srv.address)
			if err != nil {
				for _, l := range lns {
					if l == nil {
						continue
					}
					if err := l.Close(); err != nil {
						s.logger.Errorf("%s tcp listener %q close: %v", srv.label(), srv.address, err)
					}
				}
				return fmt.Errorf("%s tcp listener %q: %v", srv.label(), srv.address, err)
			}
			lns[i] = ln
		}
		if _, yes := srv.isUDP(); yes {
			addr, err := net.ResolveUDPAddr("udp", srv.address)
			if err != nil {
				return fmt.Errorf("%s resolve udp address %q: %v", srv.label(), srv.address, err)
			}
			conn, err := net.ListenUDP("udp", addr)
			if err != nil {
				return fmt.Errorf("%s udp listener %q: %v", srv.label(), srv.address, err)
			}
			conns[i] = conn
		}
	}
	for i, srv := range s.servers {
		if tcpSrv, yes := srv.isTCP(); yes {
			go func(srv *server, ln net.Listener) {
				defer s.recover()

				s.mu.Lock()
				srv.tcpAddr = ln.Addr().(*net.TCPAddr)
				s.mu.Unlock()

				s.logger.Infof("%s listening on %q", srv.label(), srv.tcpAddr.String())
				if err := tcpSrv.ServeTCP(ln); err != nil {
					s.logger.Errorf("%s serve %q: %v", srv.label(), srv.tcpAddr.String(), err)
				}
			}(srv, lns[i])
		}
		if udpSrv, yes := srv.isUDP(); yes {
			go func(srv *server, conn *net.UDPConn) {
				defer s.recover()

				s.mu.Lock()
				srv.udpAddr = conn.LocalAddr().(*net.UDPAddr)
				s.mu.Unlock()

				s.logger.Infof("%s listening on %q", srv.label(), srv.tcpAddr.String())
				if err := udpSrv.ServeUDP(conn); err != nil {
					s.logger.Errorf("%s serve %q: %v", srv.label(), srv.tcpAddr.String(), err)
				}
			}(srv, conns[i])
		}
	}
	return nil
}

// TCPAddr returns a TCP address of the listener that a server
// with a specific name is using. If there are more servers
// with the same name, the address of the first started server
// is returned.
func (s *Servers) TCPAddr(name string) (a *net.TCPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, srv := range s.servers {
		if srv.name == name {
			return srv.tcpAddr
		}
	}
	return nil
}

// UDPAddr returns a UDP address of the listener that a server
// with a specific name is using. If there are more servers
// with the same name, the address of the first started server
// is returned.
func (s *Servers) UDPAddr(name string) (a *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, srv := range s.servers {
		if srv.name == name {
			return srv.udpAddr
		}
	}
	return nil
}

// Close stops all servers, by calling Close method on each of them.
func (s *Servers) Close() {
	wg := &sync.WaitGroup{}
	for _, srv := range s.servers {
		wg.Add(1)
		go func(srv *server) {
			defer s.recover()
			defer wg.Done()

			s.logger.Infof("%s closing", srv.label())
			if err := srv.Close(); err != nil {
				s.logger.Errorf("%s close: %v", srv.label(), err)
			}
		}(srv)
	}
	wg.Wait()
}

// Shutdown gracefully stops all servers, by calling Shutdown method on each of them.
func (s *Servers) Shutdown(ctx context.Context) {
	wg := &sync.WaitGroup{}
	for _, srv := range s.servers {
		wg.Add(1)
		go func(srv *server) {
			defer s.recover()
			defer wg.Done()

			s.logger.Infof("%s shutting down", srv.label())
			if err := srv.Shutdown(ctx); err != nil {
				s.logger.Errorf("%s shutdown: %v", srv.label(), err)
			}
		}(srv)
	}
	wg.Wait()
}
