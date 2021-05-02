// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import "net/http"

// ResponseReplaceHandler calls a different Handler from a provided map of
// Handlers for HTTP Status Codes when WriteHeader is called with a specified
// status code.
func ResponseReplaceHandler(h http.Handler, handlers map[int]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(&responseReplaceWriter{w: w, r: r, handlers: handlers}, r)
	})
}

type responseReplaceWriter struct {
	w        http.ResponseWriter
	r        *http.Request
	handlers map[int]http.Handler

	stop bool
}

func (r *responseReplaceWriter) Header() http.Header {
	return r.w.Header()
}

func (r *responseReplaceWriter) Write(b []byte) (int, error) {
	if r.stop {
		return len(b), nil
	}
	return r.w.Write(b)
}

func (r *responseReplaceWriter) WriteHeader(statusCode int) {
	if h, ok := r.handlers[statusCode]; ok {
		r.stop = true
		h.ServeHTTP(r.w, r.r)
		return
	}
	r.w.WriteHeader(statusCode)
}
