// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http/httptest"
	"testing"
)

func TestGetRequestIPs(t *testing.T) {
	r := httptest.NewRequest("", "/", nil)
	r.Header.Set("X-Real-Ip", "222.12.0.3")
	r.Header.Set("X-Forwarded-For", "21.44.12.5, 67.55.9.9")

	s := "192.0.2.1, 21.44.12.5, 67.55.9.9, 222.12.0.3"

	ips := GetRequestIPs(r)

	if ips != s {
		t.Errorf("expected %s, got %s", s, ips)
	}
}

func TestGetRequestEndpoint(t *testing.T) {
	r := httptest.NewRequest("", "/", nil)

	e := GetRequestEndpoint(r)

	s := "http://example.com"

	if e != s {
		t.Errorf("expected %s, got %s", s, e)
	}
}

func TestGetRequestEndpointHTTPS(t *testing.T) {
	r := httptest.NewRequest("", "https://example.com:5555/test", nil)

	e := GetRequestEndpoint(r)

	s := "https://example.com:5555"

	if e != s {
		t.Errorf("expected %s, got %s", s, e)
	}
}

func TestGetRequestEndpointXForwardedProto(t *testing.T) {
	r := httptest.NewRequest("", "http://example.com:8000/test", nil)
	r.Header.Add("X-Forwarded-Proto", "https")

	e := GetRequestEndpoint(r)

	s := "https://example.com:8000"

	if e != s {
		t.Errorf("expected %s, got %s", s, e)
	}
}
