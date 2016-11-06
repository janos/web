// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewSetHeadersHandler(t *testing.T) {
	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()

	key := "X-Test"
	value := "1234"

	NewSetHeadersHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), map[string]string{key: value}).
		ServeHTTP(w, r)

	v := w.Header().Get(key)

	if v != value {
		t.Errorf("expected %s value for header %s, but got %s", value, key, v)
	}
}

func TestNoCacheHeadersHandler(t *testing.T) {
	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()

	NoCacheHeadersHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(w, r)

	for key, value := range noCacheHeaders {
		v := w.Header().Get(key)
		if v != value {
			t.Errorf("expected %s value for header %s, but got %s", value, key, v)
		}
	}
}

func TestNoExpireHeadersHandler(t *testing.T) {
	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()

	NoExpireHeadersHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).
		ServeHTTP(w, r)

	for key, value := range noExpireHeaders {
		v := w.Header().Get(key)
		if v != value {
			t.Errorf("expected %s value for header %s, but got %s", value, key, v)
		}
	}
}
