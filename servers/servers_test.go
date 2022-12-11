// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package servers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

var (
	errServerFailure  = errors.New("server failure")
	errServerClosed   = errors.New("server already closed")
	errServerShutdown = errors.New("server already shut down")
)

type mockServer struct {
	ln          net.Listener
	serving     chan struct{}
	didClose    bool
	didShutdown bool
	fail        bool
}

func newMockServer() *mockServer {
	return &mockServer{
		serving: make(chan struct{}),
	}
}

func (s *mockServer) ServeTCP(ln net.Listener) error {
	s.ln = ln
	s.serving <- struct{}{}
	if s.fail {
		return errServerFailure
	}
	return nil
}

func (s *mockServer) Close() error {
	if s.didClose {
		return errServerClosed
	}
	s.didClose = true
	return nil
}

func (s *mockServer) Shutdown(ctx context.Context) error {
	if s.didShutdown {
		return errServerShutdown
	}
	s.didShutdown = true
	return nil
}

type Buffer struct {
	b bytes.Buffer
	m sync.Mutex
}

func (b *Buffer) Read(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Read(p)
}

func (b *Buffer) Write(p []byte) (n int, err error) {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Write(p)
}

func (b *Buffer) String() string {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.String()
}

func (b *Buffer) Len() int {
	b.m.Lock()
	defer b.m.Unlock()
	return b.b.Len()
}

func listenerAddress(ln net.Listener) (address string) {
	a := ln.Addr().(*net.TCPAddr)
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
}

func TestEmptyServer(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	s := New()

	s.Close()

	if buf.Len() > 0 {
		t.Errorf("got %q, expected %q", buf.String(), "")
	}

	s.Shutdown(context.Background())

	if buf.Len() > 0 {
		t.Errorf("got %q, expected %q", buf.String(), "")
	}
}

func TestWithLogger(t *testing.T) {
	var buf bytes.Buffer

	s := New(WithLogger(slog.New(slog.NewJSONHandler(&buf))))

	m := newMockServer()

	s.Add("", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving

	addr := listenerAddress(m.ln)

	l := fmt.Sprintf(",\"level\":\"INFO\",\"msg\":\"listen tcp\",\"label\":\"server\",\"address\":%q}", addr)
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}

	s.Shutdown(context.Background())
}

type panicServer struct {
	serving chan struct{}
}

func newPanicServer() *panicServer {
	return &panicServer{
		serving: make(chan struct{}),
	}
}

func (s *panicServer) ServeTCP(_ net.Listener) error {
	s.serving <- struct{}{}
	panic("")
}

func (s *panicServer) Close() error {
	return nil
}

func (s *panicServer) Shutdown(ctx context.Context) error {
	return nil
}

func TestWithRecoverFunc(t *testing.T) {
	log.SetOutput(io.Discard)

	mu := &sync.Mutex{}
	recovered := false

	s := New(WithRecoverFunc(func() {
		if err := recover(); err != nil {
			mu.Lock()
			defer mu.Unlock()
			recovered = true
		}
	}))

	m := newPanicServer()

	s.Add("", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving
	time.Sleep(time.Second)

	mu.Lock()
	defer mu.Unlock()
	if !recovered {
		t.Error("not recovered from panic")
	}

	s.Shutdown(context.Background())
}

func TestServersShutdown(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	s := New()

	m := newMockServer()

	s.Add("", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving

	addr := listenerAddress(m.ln)

	l := fmt.Sprintf("INFO listen tcp label=server address=%v", addr)
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}

	s.Shutdown(context.Background())

	l = "INFO shutting down server name=server"
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}

	s.Shutdown(context.Background())

	l = "ERROR shutting down server name=server err=\"server already shut down\""
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}
}

func TestServersClose(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	s := New()

	m := newMockServer()

	s.Add("", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving

	addr := listenerAddress(m.ln)

	l := fmt.Sprintf("INFO listen tcp label=server address=%v", addr)
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}

	s.Close()

	l = "INFO closing server name=server"
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}

	s.Close()

	l = "ERROR closing server name=server err=\"server already closed\""
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}
}
func TestServerName(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	s := New()

	m := newMockServer()

	s.Add("mocked", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving

	addr := listenerAddress(m.ln)

	l := fmt.Sprintf("INFO listen tcp label=\"mocked server\" address=%v", addr)
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}

	s.Shutdown(context.Background())

	l = "INFO shutting down server name=\"mocked server\""
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}
}

func TestAddressConflict(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	ln, err := net.Listen("tcp", "")
	if err != nil {
		t.Fatal(err)
	}
	listen := ":" + strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	defer ln.Close()

	s := New()

	s.Add("", "", newMockServer())
	s.Add("", listen, newMockServer())

	if err := s.Serve(); err == nil {
		t.Fatal("expected error")
	}

	if buf.Len() > 0 {
		t.Errorf("got %q, expected %q", buf.String(), "")
	}
}

func TestServerFailure(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	s := New()

	m := newMockServer()
	m.fail = true

	s.Add("", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving

	addr := listenerAddress(m.ln)

	l := fmt.Sprintf("INFO listen tcp label=server address=%v", addr)
	if !strings.Contains(buf.String(), l) {
		t.Errorf("got %q, expected %q", buf.String(), l)
	}
}

func TestServerTCPAddr(t *testing.T) {
	var buf Buffer
	log.SetOutput(&buf)

	s := New()

	m := newMockServer()

	s.Add("mock", "", m)

	if err := s.Serve(); err != nil {
		t.Fatal(err)
	}

	<-m.serving

	a := s.TCPAddr("mock").String()
	if a != m.ln.Addr().String() {
		t.Errorf("got %q, expected %q", a, m.ln.Addr().String())
	}

	u := s.TCPAddr("unknown")
	if u != nil {
		t.Errorf("got %v, expected %v", u, nil)
	}

	s.Shutdown(context.Background())
}
