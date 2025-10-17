// Copyright (c) 2025, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"resenje.org/web"
)

// TestFlexibleFileServer_Success verifies the primary success scenarios.
func TestFlexibleFileServer_Success(t *testing.T) {
	// Arrange: A mock filesystem that the resolver can pull from.
	mockFS := fstest.MapFS{
		"test.txt": {Data: []byte("hello world"), ModTime: time.Now()},
	}

	testCases := []struct {
		name            string
		resolver        func(r *http.Request) (fs.File, http.Header, error)
		expectedBody    string
		expectedHeaders http.Header
	}{
		{
			name: "Simple file serving",
			resolver: func(r *http.Request) (fs.File, http.Header, error) {
				file, _ := mockFS.Open("test.txt")
				return file, nil, nil
			},
			expectedBody:    "hello world",
			expectedHeaders: http.Header{"Content-Type": {"text/plain; charset=utf-8"}},
		},
		{
			name: "File serving with custom headers",
			resolver: func(r *http.Request) (fs.File, http.Header, error) {
				file, _ := mockFS.Open("test.txt")
				headers := http.Header{
					"Content-Disposition": {"attachment; filename=\"custom.txt\""},
					"X-Custom-Header":     {"value123"},
				}
				return file, headers, nil
			},
			expectedBody: "hello world",
			expectedHeaders: http.Header{
				"Content-Type":        {"text/plain; charset=utf-8"},
				"Content-Disposition": {"attachment; filename=\"custom.txt\""},
				"X-Custom-Header":     {"value123"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			spy := &spyNextHandler{}
			logBuffer := new(bytes.Buffer)
			logger := slog.New(slog.NewTextHandler(logBuffer, nil))
			handler, err := web.FlexibleFileServer(tc.resolver, logger, spy)
			if err != nil {
				t.Fatalf("FlexibleFileServer() returned unexpected error: %v", err)
			}
			req := httptest.NewRequest("GET", "/any", nil)
			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			if spy.called {
				t.Error("next handler was called unexpectedly on a successful request")
			}
			if rr.Code != http.StatusOK {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
			}
			if body := rr.Body.String(); body != tc.expectedBody {
				t.Errorf("handler returned unexpected body: got %q want %q", body, tc.expectedBody)
			}
			for key, values := range tc.expectedHeaders {
				if got := rr.Header().Get(key); got != values[0] {
					t.Errorf("mismatched header %q: got %q want %q", key, got, values[0])
				}
			}
			if logBuffer.Len() > 0 {
				t.Errorf("log buffer was written to unexpectedly: %s", logBuffer.String())
			}
		})
	}
}

// TestFlexibleFileServer_ErrorHandling tests various error scenarios to ensure that
// the handler logs the error and correctly passes control to the next handler.
func TestFlexibleFileServer_ErrorHandling(t *testing.T) {
	resolverErr := errors.New("resolver access denied")

	testCases := []struct {
		name        string
		resolver    func(r *http.Request) (fs.File, http.Header, error)
		expectedLog string
	}{
		{
			name: "Resolver returns an error",
			resolver: func(r *http.Request) (fs.File, http.Header, error) {
				return nil, nil, resolverErr
			},
			expectedLog: "resolver error",
		},
		{
			name: "Resolver returns file that fails to stat",
			resolver: func(r *http.Request) (fs.File, http.Header, error) {
				return statErrorFile{}, nil, nil
			},
			expectedLog: "failed to stat file",
		},
		{
			name: "Resolver returns a non-seekable file",
			resolver: func(r *http.Request) (fs.File, http.Header, error) {
				fileContent := "cannot seek this"
				file := &nonSeekableFile{
					Reader: strings.NewReader(fileContent),
					info: staticFileInfo{
						name:    "noseek.dat",
						size:    int64(len(fileContent)),
						modTime: time.Now(),
					},
				}
				return file, nil, nil
			},
			expectedLog: "file is not seekable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			spy := &spyNextHandler{}
			logBuffer := new(bytes.Buffer)
			logger := slog.New(slog.NewTextHandler(logBuffer, nil))
			handler, err := web.FlexibleFileServer(tc.resolver, logger, spy)
			if err != nil {
				t.Fatalf("FlexibleFileServer() returned unexpected error: %v", err)
			}
			req := httptest.NewRequest("GET", "/any", nil)
			rr := httptest.NewRecorder()

			// Act
			handler.ServeHTTP(rr, req)

			// Assert
			if !spy.called {
				t.Error("next handler was not called on error")
			}
			if rr.Code != http.StatusTeapot {
				t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusTeapot)
			}
			if !strings.Contains(logBuffer.String(), tc.expectedLog) {
				t.Errorf("log message %q does not contain expected text %q", logBuffer.String(), tc.expectedLog)
			}
		})
	}
}

// TestFlexibleFileServer_ConstructorValidation ensures that the FlexibleFileServer
// function returns an error when provided with invalid (nil) arguments.
func TestFlexibleFileServer_ConstructorValidation(t *testing.T) {
	dummyResolver := func(r *http.Request) (fs.File, http.Header, error) { return nil, nil, nil }
	dummyHandler := http.NotFoundHandler()
	dummyLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name     string
		resolver func(r *http.Request) (fs.File, http.Header, error)
		next     http.Handler
		wantErr  bool
	}{
		{"Valid arguments", dummyResolver, dummyHandler, false},
		{"Error on nil resolver", nil, dummyHandler, true},
		{"Error on nil next handler", dummyResolver, nil, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			_, err := web.FlexibleFileServer(tc.resolver, dummyLogger, tc.next)

			// Assert
			if (err != nil) != tc.wantErr {
				t.Errorf("FlexibleFileServer() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// spyNextHandler is a test helper that implements http.Handler. It is used to
// verify that the middleware under test correctly calls the next handler in the
// chain when an error is encountered.
type spyNextHandler struct {
	called bool
}

// ServeHTTP records that it was called and writes a unique status code
// to allow assertions on which handler ultimately handled the request.
func (n *spyNextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n.called = true
	w.WriteHeader(http.StatusTeapot)
}

// --- Mock fs.File Implementations for Error Path Testing ---

// staticFileInfo provides a minimal implementation of fs.FileInfo for use in tests.
type staticFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (s staticFileInfo) Name() string       { return s.name }
func (s staticFileInfo) Size() int64        { return s.size }
func (s staticFileInfo) Mode() fs.FileMode  { return 0 }
func (s staticFileInfo) ModTime() time.Time { return s.modTime }
func (s staticFileInfo) IsDir() bool        { return false }
func (s staticFileInfo) Sys() any           { return nil }

// statErrorFile is a mock fs.File that always returns an error from its Stat method.
// This is used to test the error handling path when file.Stat() fails.
type statErrorFile struct{}

func (f statErrorFile) Stat() (fs.FileInfo, error) {
	return nil, errors.New("mock stat error")
}

// Close is implemented to satisfy the fs.File interface and prevent panics
// from defer file.Close() calls in the handler.
func (f statErrorFile) Close() error { return nil }

// Read is implemented to satisfy the fs.File interface.
func (f statErrorFile) Read(p []byte) (n int, err error) { return 0, io.EOF }

// nonSeekableFile is a test utility that wraps an io.Reader to implement fs.File
// but explicitly does not implement io.Seeker. This is used to test the handler's
// error path when a file cannot be seeked, as required by http.ServeContent.
type nonSeekableFile struct {
	io.Reader
	info fs.FileInfo
}

func (f *nonSeekableFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *nonSeekableFile) Close() error               { return nil }
