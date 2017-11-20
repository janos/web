// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcServer

import (
	"net"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"resenje.org/web/servers/grpc/internal/hello"
)

type server struct{}

func (s *server) Greet(ctx context.Context, in *hello.GreetRequest) (*hello.GreetResponse, error) {
	return &hello.GreetResponse{Message: "Hello, " + in.Name + "!"}, nil
}

func TestServer(t *testing.T) {
	s := New(func() *grpc.Server {
		s := grpc.NewServer()
		hello.RegisterGreeterServer(s, &server{})
		return s
	}())

	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	addr := "localhost:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		if err := s.Serve(ln); err != nil {
			panic(err)
		}
	}()

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := hello.NewGreeterClient(conn)

	name := "Gopher"

	r, err := c.Greet(context.Background(), &hello.GreetRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	want := "Hello, Gopher!"
	if r.Message != want {
		t.Errorf("got %q, expected %q", r.Message, want)
	}
}

func TestServerShutdown(t *testing.T) {
	s := New(func() *grpc.Server {
		s := grpc.NewServer()
		hello.RegisterGreeterServer(s, &server{})
		return s
	}())

	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	addr := "localhost:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		if err := s.Serve(ln); err != nil {
			if e, ok := err.(*net.OpError); !(ok && e.Op == "accept") {
				panic(err)
			}
		}
	}()

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := hello.NewGreeterClient(conn)

	name := "Gopher"

	r, err := c.Greet(context.Background(), &hello.GreetRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	want := "Hello, Gopher!"
	if r.Message != want {
		t.Errorf("got %q, expected %q", r.Message, want)
	}

	s.Shutdown(context.Background())

	r, err = c.Greet(context.Background(), &hello.GreetRequest{Name: name})
	if err.Error() != "rpc error: code = Unavailable desc = all SubConns are in TransientFailure" {
		t.Fatal(err)
	}
}

func TestServerClose(t *testing.T) {
	s := New(func() *grpc.Server {
		s := grpc.NewServer()
		hello.RegisterGreeterServer(s, &server{})
		return s
	}())

	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	addr := "localhost:" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)

	go func() {
		if err := s.Serve(ln); err != nil {
			if e, ok := err.(*net.OpError); !(ok && e.Op == "accept") {
				panic(err)
			}
		}
	}()

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := hello.NewGreeterClient(conn)

	name := "Gopher"

	r, err := c.Greet(context.Background(), &hello.GreetRequest{Name: name})
	if err != nil {
		t.Fatal(err)
	}

	want := "Hello, Gopher!"
	if r.Message != want {
		t.Errorf("got %q, expected %q", r.Message, want)
	}

	s.Close()

	r, err = c.Greet(context.Background(), &hello.GreetRequest{Name: name})
	if err.Error() != "rpc error: code = Unavailable desc = transport is closing" ||
		strings.Contains(err.Error(), "use of closed network connection") {
		t.Fatal(err)
	}
}
