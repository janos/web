// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"resenje.org/web"
)

func TestResponseStatusRecorder_noWrite(t *testing.T) {
	w := httptest.NewRecorder()

	rec := web.NewResponseStatusRecorder(w)

	if size := rec.ResponseBodySize(); size != 0 {
		t.Errorf("got %v bytes that are written as body, want 0", size)
	}
	if status := rec.Status(); status != 0 {
		t.Errorf("git status %v, want %v", status, 0)
	}
}

func TestResponseStatusRecorder_write(t *testing.T) {
	w := httptest.NewRecorder()

	rec := web.NewResponseStatusRecorder(w)

	n, err := rec.Write([]byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("got %v bytes that are written, want 2", n)
	}
	if size := rec.ResponseBodySize(); size != 2 {
		t.Errorf("got %v bytes that are written as body, want 2", size)
	}
	if status := rec.Status(); status != http.StatusOK {
		t.Errorf("git status %v, want %v", status, http.StatusOK)
	}

	n, err = rec.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("got %v bytes that are written, want 5", n)
	}
	if size := rec.ResponseBodySize(); size != 7 {
		t.Errorf("got %v bytes that are written as body, want 7", size)
	}
}

func TestResponseStatusRecorder_writeHeader(t *testing.T) {
	w := httptest.NewRecorder()

	rec := web.NewResponseStatusRecorder(w)

	rec.WriteHeader(http.StatusTeapot)

	if size := rec.ResponseBodySize(); size != 0 {
		t.Errorf("got %v bytes that are written as body, want 0", size)
	}
	if status := rec.Status(); status != http.StatusTeapot {
		t.Errorf("git status %v, want %v", status, http.StatusTeapot)
	}
}

func TestResponseStatusRecorder_writeHeaderAfterWrite(t *testing.T) {
	w := httptest.NewRecorder()

	rec := web.NewResponseStatusRecorder(w)

	n, err := rec.Write([]byte("hi"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("got %v bytes that are written, want 2", n)
	}
	if size := rec.ResponseBodySize(); size != 2 {
		t.Errorf("got %v bytes that are written as body, want 2", size)
	}
	if status := rec.Status(); status != http.StatusOK {
		t.Errorf("git status %v, want %v", status, http.StatusOK)
	}

	rec.WriteHeader(http.StatusTeapot)

	if status := rec.Status(); status != http.StatusOK {
		t.Errorf("git status %v, want %v", status, http.StatusOK)
	}
}
