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

func TestHandleMethods_KnownMethod(t *testing.T) {
	body := "got post"
	methods := map[string]http.Handler{
		"POST": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(body))
		}),
	}
	contentType := "text/plain"
	r := httptest.NewRequest("POST", "/", nil)
	w := httptest.NewRecorder()

	HandleMethods(methods, body, contentType, w, r)

	statusCode := w.Code
	if statusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, statusCode)
	}
	v := w.Body.String()
	if v != body {
		t.Errorf("expected body %q, got %q", body, v)
	}
}

func TestHandleMethods_UnknownMethod(t *testing.T) {
	methods := map[string]http.Handler{
		"POST": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	body := http.StatusText(http.StatusMethodNotAllowed)
	contentType := "text/plain"
	allow := "POST"
	r := httptest.NewRequest("", "/", nil)
	w := httptest.NewRecorder()

	HandleMethods(methods, body, contentType, w, r)

	statusCode := w.Code
	if statusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status code %d, got %d", http.StatusMethodNotAllowed, statusCode)
	}
	v := w.Body.String()
	if v != body+"\n" {
		t.Errorf("expected body %q, got %q", body+"\n", v)
	}
	v = w.Header().Get("Allow")
	if v != allow {
		t.Errorf("expected Allow header %q, got %q", allow, v)
	}
}

func TestHandleMethods_Options(t *testing.T) {
	methods := map[string]http.Handler{
		"POST": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	body := http.StatusText(http.StatusMethodNotAllowed)
	contentType := "text/plain"
	allow := "POST"
	r := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	HandleMethods(methods, body, contentType, w, r)

	statusCode := w.Code
	if statusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, statusCode)
	}
	v := w.Header().Get("Allow")
	if v != allow {
		t.Errorf("expected Allow header %q, got %q", allow, v)
	}
}
