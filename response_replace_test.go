// Copyright (c) 2021, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"resenje.org/web"
)

func TestResponseReplaceHanlder(t *testing.T) {
	handler := web.ResponseReplaceHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprint(w, "OK")
			return
		}
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "sensitive information")
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "this should never be seen")
	}), map[int]http.Handler{
		http.StatusNotFound: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound) // keep the same status
			fmt.Fprint(w, "the page is not here")
		}),
		http.StatusInternalServerError: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot) // set a different status
			fmt.Fprint(w, "have some tea")
		}),
	})

	t.Run("no replace", func(t *testing.T) {
		r := httptest.NewRequest("", "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		assertResponse(t, w, http.StatusOK, "OK")
	})

	t.Run("replace with the same status code", func(t *testing.T) {
		r := httptest.NewRequest("", "/password", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		assertResponse(t, w, http.StatusNotFound, "the page is not here")
	})

	t.Run("replace with a different status code", func(t *testing.T) {
		r := httptest.NewRequest("", "/broken", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, r)

		assertResponse(t, w, http.StatusTeapot, "have some tea")
	})
}

func assertResponse(t *testing.T, r *httptest.ResponseRecorder, wantStatusCode int, wantBody string) {
	gotStatusCode := r.Result().StatusCode
	if gotStatusCode != wantStatusCode {
		t.Errorf("got status code %v, want %v", gotStatusCode, wantStatusCode)
	}
	gotBody := r.Body.String()
	if gotBody != wantBody {
		t.Errorf("got body %q, want %q", gotBody, wantBody)
	}
}
