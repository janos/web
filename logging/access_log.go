// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"golang.org/x/exp/slog"
)

type AccessLogOptions struct {
	RealIPHeaderName string
	PreHook          http.HandlerFunc
	PostHook         func(code int, duration time.Duration, written int64)
	LogMessage       string
}

// NewHandler returns a handler that logs HTTP requests.
// It logs information about remote address, X-Forwarded-For or X-Real-Ip,
// HTTP method, request URI, HTTP protocol, HTTP response status, total bytes
// written to http.ResponseWriter, response duration, HTTP referrer and
// HTTP client user agent.
func NewAccessLogHandler(h http.Handler, logger *slog.Logger, o *AccessLogOptions) http.Handler {
	if o == nil {
		o = new(AccessLogOptions)
	}
	realIPheaders := []string{
		"X-Forwarded-For",
		"X-Real-Ip",
	}
	if o.RealIPHeaderName != "" && o.RealIPHeaderName != "X-Forwarded-For" && o.RealIPHeaderName != "X-Real-Ip" {
		realIPheaders = append(realIPheaders, o.RealIPHeaderName)
	}
	logMessage := o.LogMessage
	if logMessage == "" {
		logMessage = "access"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o.PreHook != nil {
			o.PreHook(w, r)
		}

		m := httpsnoop.CaptureMetrics(h, w, r)

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		ips := []string{ip}
		for _, key := range realIPheaders {
			if v := r.Header.Get(key); v != "" {
				ips = append(ips, v)
			}
		}

		status := m.Code

		attrs := []slog.Attr{
			slog.String("remote address", r.RemoteAddr),
			slog.String("ips", strings.Join(ips, ", ")),
			slog.String("method", r.Method),
			slog.String("uri", r.RequestURI),
			slog.String("proto", r.Proto),
			slog.Int("status", status),
			slog.Int64("response size", m.Written),
			slog.String("duration", m.Duration.String()),
		}

		if referrer := r.Referer(); referrer != "" {
			attrs = append(attrs, slog.String("referer", referrer))
		}
		if userAgent := r.UserAgent(); userAgent != "" {
			attrs = append(attrs, slog.String("user agent", userAgent))
		}

		var level slog.Level
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		case status >= 300:
			level = slog.LevelInfo
		case status >= 200:
			level = slog.LevelInfo
		default:
			level = slog.LevelDebug
		}

		logger.LogAttrs(r.Context(), level, logMessage, attrs...)

		if o.PostHook != nil {
			o.PostHook(m.Code, m.Duration, m.Written)
		}
	})
}
