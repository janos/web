// Copyright (c) 2018, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoundTripperFunc(t *testing.T) {
	header := "a clockwork orange"
	rt := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		r.Header.Set("X-Header", header)
		return nil, nil
	})

	r, err := http.NewRequest(http.MethodGet, "http://localhost:8080", nil)
	if err != nil {
		t.Error(err)
	}
	if _, err := rt.RoundTrip(r); err != nil {
		t.Error(err)
	}

	got := r.Header.Get("X-Header")
	if got != header {
		t.Errorf("got header X-Header %q, expected %q", got, header)
	}
}

func TestRoundTripperFuncInClient(t *testing.T) {
	userAgent := "a clockwork orange"
	client := &http.Client{
		Transport: RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.Header.Set("User-Agent", userAgent)
			return http.DefaultTransport.RoundTrip(r)
		}),
	}

	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
	}))
	defer server.Close()

	_, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if got != userAgent {
		t.Errorf("got header User-Agent %q, expected %q", got, userAgent)
	}
}
