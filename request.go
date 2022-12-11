// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net"
	"net/http"
	"strings"
)

// GetRequestIPs returns all possible IPs found in HTTP request.
func GetRequestIPs(r *http.Request, realIPHeaders ...string) string {
	if realIPHeaders == nil {
		realIPHeaders = []string{"X-Forwarded-For", "X-Real-Ip"}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	ips := []string{ip}
	for _, key := range realIPHeaders {
		v := r.Header.Get(key)
		if v != "" {
			ips = append(ips, v)
		}
	}
	return strings.Join(ips, ", ")
}

// GetRequestEndpoint returns request's host perpended with protocol:
// protocol://host.
func GetRequestEndpoint(r *http.Request) string {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		if r.TLS == nil {
			proto = "http"
		} else {
			proto = "https"
		}
	}
	return proto + "://" + r.Host
}
