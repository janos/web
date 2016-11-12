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

func TestDefaultNotFoundHandler(t *testing.T) {
	w := httptest.NewRecorder()
	DefaultNotFoundHandler(w, nil)

	code := w.Result().StatusCode
	if code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, code)
	}
}

func TestDefaultForbiddenHandler(t *testing.T) {
	w := httptest.NewRecorder()
	DefaultForbiddenHandler(w, nil)

	code := w.Result().StatusCode
	if code != http.StatusForbidden {
		t.Errorf("expected status code %d, got %d", http.StatusForbidden, code)
	}
}

func TestDefaultInternalServerErrorHandler(t *testing.T) {
	w := httptest.NewRecorder()
	DefaultInternalServerErrorHandler(w, nil)

	code := w.Result().StatusCode
	if code != http.StatusInternalServerError {
		t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, code)
	}
}
