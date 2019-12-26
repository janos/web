// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package accesslog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"resenje.org/logging"
)

type Formatter struct{}

func (formatter *Formatter) Format(record *logging.Record) string {
	return fmt.Sprintf("%s %s", record.Level, record.Message)
}

func TestAccessLog(t *testing.T) {
	for _, tc := range []struct {
		name       string
		request    *http.Request
		statusCode int
		pattern    *regexp.Regexp
	}{
		{
			name:    "GET",
			request: httptest.NewRequest("", "/", nil),
			pattern: regexp.MustCompile(`^INFO 192.0.2.1:1234 "-" "GET / HTTP/1.1" 200 9 0.\d{6} "-" "-"$`),
		},
		{
			name:       "POST",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: http.StatusOK,
			pattern:    regexp.MustCompile(`^INFO 192.0.2.1:1234 "-" "POST / HTTP/1.1" 200 9 0.\d{6} "-" "-"$`),
		},
		{
			name: "XForwardedFor",
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/", nil)
				r.Header.Set("X-Forwarded-For", "1.1.1.1, 1.2.2.2")
				return r
			}(),
			statusCode: http.StatusOK,
			pattern:    regexp.MustCompile(`^INFO 192.0.2.1:1234 "1.1.1.1, 1.2.2.2" "POST / HTTP/1.1" 200 9 0.\d{6} "-" "-"$`),
		},
		{
			name: "XRealIp",
			request: func() *http.Request {
				r := httptest.NewRequest("POST", "/", nil)
				r.Header.Set("X-Real-Ip", "1.2.3.3")
				return r
			}(),
			statusCode: http.StatusOK,
			pattern:    regexp.MustCompile(`^INFO 192.0.2.1:1234 "1.2.3.3" "POST / HTTP/1.1" 200 9 0.\d{6} "-" "-"$`),
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
			pattern:    regexp.MustCompile(`^INFO 192.0.2.1:1234 "1.1.1.1, 1.2.2.2, 1.2.3.3" "POST / HTTP/1.1" 200 9 0.\d{6} "-" "-"$`),
		},
		{
			name:       "100",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 100,
			pattern:    regexp.MustCompile(`^DEBUG 192.0.2.1:1234 "-" "POST / HTTP/1.1" 100 9 0.\d{6} "-" "-"$`),
		},
		{
			name:       "300",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 300,
			pattern:    regexp.MustCompile(`^INFO 192.0.2.1:1234 "-" "POST / HTTP/1.1" 300 9 0.\d{6} "-" "-"$`),
		},
		{
			name:       "400",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 400,
			pattern:    regexp.MustCompile(`^WARNING 192.0.2.1:1234 "-" "POST / HTTP/1.1" 400 9 0.\d{6} "-" "-"$`),
		},
		{
			name:       "500",
			request:    httptest.NewRequest("POST", "/", nil),
			statusCode: 500,
			pattern:    regexp.MustCompile(`^ERROR 192.0.2.1:1234 "-" "POST / HTTP/1.1" 500 9 0.\d{6} "-" "-"$`),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			logHander := &logging.MemoryHandler{Formatter: &Formatter{}, Level: logging.DEBUG}
			NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte("test data"))
			}), logging.NewLogger("test", logging.DEBUG, []logging.Handler{logHander}, 0)).ServeHTTP(w, tc.request)

			logging.WaitForAllUnprocessedRecords()

			got := logHander.Messages[0]
			if !tc.pattern.MatchString(got) {
				t.Errorf("%s did not match pattern", got)
			}
		})
	}

}
