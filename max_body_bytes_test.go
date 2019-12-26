// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMaxBodyBytesHandler_Pass(t *testing.T) {
	h := MaxBodyBytesHandler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	}
	r := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Body.String() != "ok" {
		t.Errorf("expected response body %q, got %q", "ok", w.Body.String())
	}
}

func TestMaxBodyBytesHandler_Block(t *testing.T) {
	h := MaxBodyBytesHandler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.Copy(ioutil.Discard, r.Body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
			}
		}),
		Limit: 10,
		BodyFunc: func(r *http.Request) (body string, err error) {
			return http.StatusText(http.StatusRequestEntityTooLarge), nil
		},
		ContentType: "text/plain",
	}
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte("12345678901")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	b, err := ioutil.ReadAll(w.Result().Body)
	if err != nil {
		t.Error(err)
	}

	if string(b) != http.StatusText(http.StatusRequestEntityTooLarge)+"\n" {
		t.Errorf("expected response body %q, got %q", http.StatusText(http.StatusRequestEntityTooLarge)+"\n", string(b))
	}

	contentType := w.Result().Header.Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("expected response Content-Type header %q, got %q", "text/plain", contentType)
	}
}

func TestMaxBodyBytesHandler_PanicError(t *testing.T) {
	h := MaxBodyBytesHandler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.Copy(ioutil.Discard, r.Body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
			}
		}),
		Limit: 10,
		BodyFunc: func(r *http.Request) (body string, err error) {
			return "", errTestMaxBodyBytesHandler
		},
	}
	defer func() {
		if err := recover(); err != errTestMaxBodyBytesHandler {
			t.Errorf("expected %v error, got %v", errTestMaxBodyBytesHandler, err)
		}
	}()

	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte("12345678901")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
}

var errTestMaxBodyBytesHandler = errors.New("test")

func TestMaxBodyBytesHandler_CustomError(t *testing.T) {
	var e error
	h := MaxBodyBytesHandler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.Copy(ioutil.Discard, r.Body); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(err.Error()))
			}
		}),
		Limit: 10,
		BodyFunc: func(r *http.Request) (body string, err error) {
			return "", errTestMaxBodyBytesHandler
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			e = err
		},
	}
	r := httptest.NewRequest("POST", "/", bytes.NewReader([]byte("12345678901")))
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if e != errTestMaxBodyBytesHandler {
		t.Errorf("expected %v error, got %v", errTestMaxBodyBytesHandler, e)
	}
}
