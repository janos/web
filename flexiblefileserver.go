// Copyright (c) 2025, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
)

// FlexibleFileServer creates and returns a new http.Handler middleware.
// The resolver function is responsible for finding and opening the file to be
// served, returning an fs.File handle. The middleware takes ownership of the
// returned fs.File and is responsible for closing it.
// If any step fails, the error is logged and the request is passed to the next handler.
func FlexibleFileServer(resolver func(r *http.Request) (file fs.File, headers http.Header, err error), logger *slog.Logger, next http.Handler) (http.Handler, error) {
	if resolver == nil {
		return nil, errors.New("flexiblefs: resolver cannot be nil")
	}
	if next == nil {
		return nil, errors.New("flexiblefs: next handler cannot be nil")
	}

	if logger == nil {
		// Default to a logger that discards all output if none is provided.
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &fileServer{
		resolver: resolver,
		logger:   logger,
		next:     next,
	}, nil
}

// fileServer is an http.Handler that serves files provided by its resolver function.
// It acts as middleware, calling the next handler if it encounters an error.
type fileServer struct {
	resolver func(r *http.Request) (file fs.File, headers http.Header, err error)
	logger   *slog.Logger
	next     http.Handler
}

// ServeHTTP uses the resolver to get a file handle. If successful, it serves
// the file using http.ServeContent. If any error occurs, it logs the error
// and calls the next handler in the chain. It takes ownership of the fs.File
// returned by the resolver and ensures it is closed.
func (h *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Use the resolver to get the file handle and custom headers.
	file, customHeaders, err := h.resolver(r)
	if err != nil {
		h.logger.Error("resolver error", "method", r.Method, "url", r.URL.String(), "error", err)
		h.next.ServeHTTP(w, r)
		return
	}
	// The handler takes ownership of the file and guarantees it will be closed.
	defer func() { _ = file.Close() }()

	// 2. Get file stats to provide to ServeContent.
	stat, err := file.Stat()
	if err != nil {
		h.logger.Error("failed to stat file", "method", r.Method, "url", r.URL.String(), "error", err)
		h.next.ServeHTTP(w, r)
		return
	}

	// 3. Apply custom headers returned by the resolver before serving content.
	for key, values := range customHeaders {
		// Using Add, not Set, to allow for multiple values for the same header key.
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 4. An fs.File is not guaranteed to implement io.Seeker.
	// http.ServeContent requires an io.ReadSeeker. We must check for this.
	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		h.logger.Error("file is not seekable", "method", r.Method, "url", r.URL.String())
		h.next.ServeHTTP(w, r)
		return
	}

	// 5. Use http.ServeContent to handle the actual serving.
	// This function correctly handles Range requests (for partial content),
	// sets the ETag header, and manages If-Modified-Since caching headers.
	// Passing an empty string for the name triggers Content-Type sniffing.
	http.ServeContent(w, r, "", stat.ModTime(), seeker)
}
