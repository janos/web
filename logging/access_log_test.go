// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"resenje.org/web/logging"
)

func TestAccessLog(t *testing.T) {
	for _, tc := range []struct {
		name       string
		request    *http.Request
		statusCode int
		pattern    string
	}{
		{
			name:       "GET",
			request:    httptest.NewRequest("", "/", nil),
			statusCode: http.StatusOK,
			pattern:    `level=INFO msg=access "remote address"=192.0.2.1:1234 ips=192.0.2.1 method=GET uri=/ proto=HTTP/1.1 status=200 "response size"=9 duration=`,
		},
		{
			name:       "POST",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: http.StatusOK,
			pattern:    `level=INFO msg=access "remote address"=192.0.2.1:1234 ips=192.0.2.1 method=POST uri=/ proto=HTTP/1.1 status=200 "response size"=9 duration=`,
		},
		{
			name: "XForwardedFor",
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/", nil)
				r.Header.Set("X-Forwarded-For", "1.1.1.1, 1.2.2.2")
				return r
			}(),
			statusCode: http.StatusOK,
			pattern:    `level=INFO msg=access "remote address"=192.0.2.1:1234 ips="192.0.2.1, 1.1.1.1, 1.2.2.2" method=POST uri=/ proto=HTTP/1.1 status=200 "response size"=9 duration=`,
		},
		{
			name: "XRealIp",
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/", nil)
				r.Header.Set("X-Real-Ip", "1.2.3.3")
				return r
			}(),
			statusCode: http.StatusOK,
			pattern:    `level=INFO msg=access "remote address"=192.0.2.1:1234 ips="192.0.2.1, 1.2.3.3" method=POST uri=/ proto=HTTP/1.1 status=200 "response size"=9 duration=`,
		},
		{
			name: "XForwardedForAndXRealIp",
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/", nil)
				r.Header.Set("X-Forwarded-For", "1.1.1.1, 1.2.2.2")
				r.Header.Set("X-Real-Ip", "1.2.3.3")
				return r
			}(),
			statusCode: http.StatusOK,
			pattern:    `level=INFO msg=access "remote address"=192.0.2.1:1234 ips="192.0.2.1, 1.1.1.1, 1.2.2.2, 1.2.3.3" method=POST uri=/ proto=HTTP/1.1 status=200 "response size"=9 duration=`,
		},
		{
			name:       "300",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 300,
			pattern:    `level=INFO msg=access "remote address"=192.0.2.1:1234 ips=192.0.2.1 method=POST uri=/ proto=HTTP/1.1 status=300 "response size"=9 duration=`,
		},
		{
			name:       "400",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 400,
			pattern:    `level=WARN msg=access "remote address"=192.0.2.1:1234 ips=192.0.2.1 method=POST uri=/ proto=HTTP/1.1 status=400 "response size"=9 duration=`,
		},
		{
			name:       "500",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 500,
			pattern:    `level=ERROR msg=access "remote address"=192.0.2.1:1234 ips=192.0.2.1 method=POST uri=/ proto=HTTP/1.1 status=500 "response size"=9 duration=`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			var buf bytes.Buffer

			logging.NewAccessLogHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte("test data"))
			}), slog.New(slog.NewTextHandler(&buf, nil)), nil).ServeHTTP(w, tc.request)

			got := buf.String()
			if !strings.Contains(got, tc.pattern) {
				t.Errorf("got %v, want %v", got, tc.pattern)
			}
		})
	}
}
