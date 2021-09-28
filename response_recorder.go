// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import "net/http"

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
