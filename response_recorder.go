// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
)

// ResponseStatusRecorder implements http.ResponseWriter that keeps tack of HTTP
// response status code and written body size in bytes.
type ResponseStatusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

// NewResponseStatusRecorder wraps an http.ResponseWriter with
// ResponseStatusRecorder in order to record the status code and written body
// size.
func NewResponseStatusRecorder(w http.ResponseWriter) *ResponseStatusRecorder {
	return &ResponseStatusRecorder{
		ResponseWriter: w,
	}
}

// Write implements http.ResponseWriter.
func (r *ResponseStatusRecorder) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	if err != nil {
		return 0, err
	}
	if r.status == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		r.status = http.StatusOK
	}
	r.size += size
	return size, err
}

// WriteHeader implements http.ResponseWriter.
func (r *ResponseStatusRecorder) WriteHeader(s int) {
	r.ResponseWriter.WriteHeader(s)
	if r.status == 0 {
		r.status = s
	}
}

// Status returns the responded status code. If it is 0, no response data has
// been written.
func (r *ResponseStatusRecorder) Status() int {
	return r.status
}

// ResponseBodySize returns the number of bytes that are written as the response body.
func (r *ResponseStatusRecorder) ResponseBodySize() int {
	return r.size
}

func (r *ResponseStatusRecorder) Flush() {
	f, ok := r.ResponseWriter.(http.Flusher)
	if !ok {
		return
	}
	f.Flush()
}

func (r *ResponseStatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not implement http.Hijacker")
	}
	return h.Hijack()
}

func (r *ResponseStatusRecorder) ReadFrom(src io.Reader) (int64, error) {
	rf, ok := r.ResponseWriter.(io.ReaderFrom)
	if !ok {
		return 0, errors.New("response writer does not implement io.ReaderFrom")
	}
	return rf.ReadFrom(src)
}

func (r *ResponseStatusRecorder) Push(target string, opts *http.PushOptions) error {
	p, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return errors.New("response writer does not implement http.Pusher")
	}
	return p.Push(target, opts)
}
