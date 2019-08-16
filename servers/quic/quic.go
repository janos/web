// Copyright (c) 2018, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package quicServer

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/lucas-clemente/quic-go/http3"
	"resenje.org/web/servers"
)

var (
	_ servers.Server    = new(Server)
	_ servers.UDPServer = new(Server)
)

// Options struct holds parameters that can be configure using
// functions with prefix With.
type Options struct {
	tlsConfig *tls.Config
}

// Option is a function that sets optional parameters for
// the Server.
type Option func(*Options)

// WithTLSConfig sets a TLS configuration for the HTTP server
// and creates a TLS listener.
func WithTLSConfig(tlsConfig *tls.Config) Option { return func(o *Options) { o.tlsConfig = tlsConfig } }

// Server wraps http3.Server to provide methods for
// resenje.org/web/servers.Server interface.
type Server struct {
	*http3.Server
}

// New creates a new instance of Server.
func New(handler http.Handler, opts ...Option) (s *Server) {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	s = &Server{
		Server: &http3.Server{
			Server: &http.Server{
				Handler:   handler,
				TLSConfig: o.tlsConfig,
			},
		},
	}
	return
}

// ServeUDP serves requests over UDP connection.
func (s *Server) ServeUDP(conn *net.UDPConn) (err error) {
	s.Server.Server.Addr = conn.LocalAddr().String()
	return s.Server.Serve(conn)
}

// Shutdown calls http3.Server.Close method.
func (s *Server) Shutdown(_ context.Context) (err error) {
	return s.Server.Close()
}

// QuicHeadersHandler should be used as a middleware to set
// quic related headers to TCP server that suggest alternative svc.
func (s *Server) QuicHeadersHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.SetQuicHeaders(w.Header())
		h.ServeHTTP(w, r)
	})
}
