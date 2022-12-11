// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewStaticFilesHandler(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}
	dir := t.TempDir()

	content := "file content"

	f, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	r := httptest.NewRequest("", "/static/"+fn, nil)
	w := httptest.NewRecorder()

	NewStaticFilesHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected call to handler")
	}), "/static", http.Dir(dir)).ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestNewStaticFilesHandlerMissingFile(t *testing.T) {
	dir := t.TempDir()

	called := false

	r := httptest.NewRequest("", "/static/no-file", nil)
	w := httptest.NewRecorder()
	NewStaticFilesHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}), "/static", http.Dir(dir)).ServeHTTP(w, r)

	if !called {
		t.Error("handler was not called")
	}
}
