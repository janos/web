// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"resenje.org/jsonhttp"

	"resenje.org/web"
	"resenje.org/web/recovery"
)

func newRedirectDomainHandler(domain, httpsPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, port, _ := net.SplitHostPort(r.Host)
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if fs := r.Header.Get("X-Forwarded-Proto"); fs != "" {
			scheme = strings.ToLower(fs)
		}
		if httpsPort != "" {
			if scheme != "https" && port != httpsPort {
				scheme = "https"
				port = httpsPort
			}
		}
		switch {
		case scheme == "http" && port == "80":
			port = ""
		case scheme == "https" && port == "443":
			port = ""
		case port == "":
		default:
			port = ":" + port
		}
		http.Redirect(w, r, strings.Join([]string{scheme, "://", domain, port, r.RequestURI}, ""), http.StatusMovedPermanently)
	})
}

func redirectHTTPSHandler(h http.Handler, httpsPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = r.Host
			}
			newRedirectDomainHandler(host, httpsPort).ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func textNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, http.StatusText(http.StatusNotFound))
}

// statusResponse is a response of a status API handler.
type statusResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

func (s *Server) statusAPIHandler(w http.ResponseWriter, r *http.Request) {
	jsonhttp.OK(w, statusResponse{
		Name:    s.name,
		Version: s.Version(),
		Uptime:  time.Since(s.startTime).String(),
	})
}

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s version %s, uptime %s", s.name, s.Version(), time.Since(s.startTime))
}

// Recovery handler for JSON API routers.
func (s *Server) jsonRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(s.Version()),
		recovery.WithLogger(s.logger),
		recovery.WithNotifier(s.emailService),
		recovery.WithPanicResponse(`{"message":"Internal Server Error","code":500}`, "application/json; charset=utf-8"),
	)
}

// Recovery handler for JSON API routers.
func (s *Server) textRecoveryHandler(h http.Handler) http.Handler {
	return recovery.New(h,
		recovery.WithLabel(s.Version()),
		recovery.WithLogger(s.logger),
		recovery.WithNotifier(s.emailService),
		recovery.WithPanicResponse("Internal Server Error", "text/plain; charset=utf-8"),
	)
}

type jsonMethodHandler map[string]http.Handler

func (h jsonMethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	web.HandleMethods(h, `{"message":"Method Not Allowed","code":405}`, "application/json; charset=utf-8", w, r)
}
