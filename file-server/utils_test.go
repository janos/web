// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileServer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirect(t *testing.T) {
	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()

	location := "http://localhost/test"
	redirect(w, r, location)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	cc := w.Result().Header.Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected Cache-Control header %q, got %q", "no-cache", cc)
	}

	loc := w.Result().Header.Get("Location")
	if loc != location {
		t.Errorf("expected Location header %q, got %q", location, loc)
	}
}

func TestRedirectWithQuery(t *testing.T) {
	r := httptest.NewRequest("", "/?test=123", nil)
	w := httptest.NewRecorder()

	location := "http://localhost/test"
	redirect(w, r, location)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	cc := w.Result().Header.Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("expected Cache-Control header %q, got %q", "no-cache", cc)
	}

	loc := w.Result().Header.Get("Location")
	if loc != location+"?test=123" {
		t.Errorf("expected Location header %q, got %q", location+"?test=123", loc)
	}
}

func TestOpen(t *testing.T) {
	f, err := open("", "utils_test.go", nil)
	if err != nil {
		t.Error(err)
	}
	if f == nil {
		t.Error("expected file object, got nil")
	}
}
