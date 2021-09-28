// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package accesslog

import (
	"net/http"
	"strings"
	"time"

	"resenje.org/logging"
	"resenje.org/web"
)

// NewHandler returns a handler that logs HTTP requests.
// It logs information about remote address, X-Forwarded-For or X-Real-Ip,
// HTTP method, request URI, HTTP protocol, HTTP response status, total bytes
// written to http.ResponseWriter, response duration, HTTP referrer and
// HTTP client user agent.
func NewHandler(h http.Handler, logger *logging.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		rl := web.NewResponseStatusRecorder(w)
		h.ServeHTTP(rl, r)
		referrer := r.Referer()
		if referrer == "" {
			referrer = "-"
		}
		userAgent := r.UserAgent()
		if userAgent == "" {
			userAgent = "-"
		}
		ips := []string{}
		xfr := r.Header.Get("X-Forwarded-For")
		if xfr != "" {
			ips = append(ips, xfr)
		}
		xri := r.Header.Get("X-Real-Ip")
		if xri != "" {
			ips = append(ips, xri)
		}
		xips := "-"
		if len(ips) > 0 {
			xips = strings.Join(ips, ", ")
		}
		status := rl.Status()
		var level logging.Level
		switch {
		case status >= 500:
			level = logging.ERROR
		case status >= 400:
			level = logging.WARNING
		case status >= 300:
			level = logging.INFO
		case status >= 200:
			level = logging.INFO
		default:
			level = logging.DEBUG
		}
		logger.Logf(level, "%s \"%s\" \"%v %s %v\" %d %d %f \"%s\" \"%s\"", r.RemoteAddr, xips, r.Method, r.RequestURI, r.Proto, status, rl.ResponseBodySize(), time.Since(startTime).Seconds(), referrer, userAgent)
	})
}
