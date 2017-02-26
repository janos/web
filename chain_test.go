// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httputils

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChain(t *testing.T) {
	handlers := []func(http.Handler) http.Handler{
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("0"))
				if h != nil {
					h.ServeHTTP(w, r)
				}
			})
		},
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("1"))
				if h != nil {
					h.ServeHTTP(w, r)
				}
			})
		},
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("2"))
				if h != nil {
					h.ServeHTTP(w, r)
				}
			})
		},
	}

	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()
	ChainHandlers(handlers...).ServeHTTP(w, r)

	b, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Error(err)
	}
	if string(b) != "012" {
		t.Errorf("expected body %q, got %q", "012", string(b))
	}
}

func TestFinalHandler(t *testing.T) {
	handlers := []func(http.Handler) http.Handler{
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("0"))
				if h != nil {
					h.ServeHTTP(w, r)
				}
			})
		},
		FinalHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("1"))
		})),
	}

	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()
	ChainHandlers(handlers...).ServeHTTP(w, r)

	b, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Error(err)
	}
	if string(b) != "01" {
		t.Errorf("expected body %q, got %q", "01", string(b))
	}
}

func TestFinalHandlerFunc(t *testing.T) {
	handlers := []func(http.Handler) http.Handler{
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("0"))
				if h != nil {
					h.ServeHTTP(w, r)
				}
			})
		},
		FinalHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("1"))
		}),
	}

	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()
	ChainHandlers(handlers...).ServeHTTP(w, r)

	b, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Error(err)
	}
	if string(b) != "01" {
		t.Errorf("expected body %q, got %q", "01", string(b))
	}
}
