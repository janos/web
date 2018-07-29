// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcServer

import (
	"context"
	"net"

	"google.golang.org/grpc"
	"resenje.org/web/servers"
)

var (
	_ servers.Server    = new(Server)
	_ servers.TCPServer = new(Server)
)

// Server wraps grpc.Server to provide methods for
// resenje.org/web/servers.Server interface.
type Server struct {
	*grpc.Server
}

// New creates a new instance of Server.
func New(server *grpc.Server) (s *Server) {
	return &Server{
		Server: server,
	}
}

// ServeTCP serves request on TCP listener.
func (s *Server) ServeTCP(ln net.Listener) (err error) {
	return s.Server.Serve(ln)
}

// Close executes grpc.Server.Stop method.
func (s *Server) Close() (err error) {
	s.Server.Stop()
	return
}

// Shutdown executes grpc.Server.GracefulStop method.
func (s *Server) Shutdown(ctx context.Context) (err error) {
	s.Server.GracefulStop()
	return
}
