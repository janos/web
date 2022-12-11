// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileServer

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServer(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	r := httptest.NewRequest("", "/assets/"+fn, nil)
	w := httptest.NewRecorder()

	New("/assets", dir, nil).ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerAltDir(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.CreateTemp(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	altDir, err := os.MkdirTemp("", "file-server-test-alt")
	if err != nil {
		t.Error(err)
	}
	defer os.RemoveAll(dir)

	content = "file alt content"

	err = os.WriteFile(filepath.Join(altDir, fn), []byte(content), 0666)
	if err != nil {
		t.Error(err)
	}

	r := httptest.NewRequest("", "/assets/"+fn, nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		AltDir: altDir,
	}).ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerFileNotFound(t *testing.T) {
	dir := t.TempDir()

	r := httptest.NewRequest("", "/assets/missing-file", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, nil).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, code)
	}
}

func TestServerFileNotFoundCustom(t *testing.T) {
	dir := t.TempDir()

	r := httptest.NewRequest("", "/assets/missing-file", nil)
	w := httptest.NewRecorder()

	content := "Test"
	New("/assets", dir, &Options{
		NotFoundHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, content, http.StatusTeapot)
		}),
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusTeapot {
		t.Errorf("expected status code %d, got %d", http.StatusTeapot, code)
	}

	body := w.Body.String()
	if body != content+"\n" {
		t.Errorf("expected content %q, got %q", content+"\n", body)
	}
}

func TestServerDirNotFound(t *testing.T) {
	dir := t.TempDir()

	r := httptest.NewRequest("", "/assets", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, nil).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, code)
	}
}

func TestServerServeIndexPage(t *testing.T) {
	dir := t.TempDir()

	content := "index"

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		IndexPage: "index.html",
	}).ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerRedirectIndexPage(t *testing.T) {
	dir := t.TempDir()

	content := "index"

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets/index.html", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		IndexPage: "index.html",
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "./" {
		t.Errorf("expected Location header %q, got %q", "./", loc)
	}
}

func TestServerRedirectTrailingSlashDir(t *testing.T) {
	dir := t.TempDir()

	content := "index"

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		IndexPage:             "index.html",
		RedirectTrailingSlash: true,
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "/assets/" {
		t.Errorf("expected Location header %q, got %q", "/assets/", loc)
	}
}

func TestServerRedirectTrailingSlashDirFile(t *testing.T) {
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

	r := httptest.NewRequest("", "/assets/"+fn+"/", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		IndexPage:             "index.html",
		RedirectTrailingSlash: true,
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "../"+fn {
		t.Errorf("expected Location header %q, got %q", "../"+fn, loc)
	}
}

func TestServerHasher(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	h := New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	})

	r := httptest.NewRequest("", "/assets/"+fn+".d10b4c3f", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerHasherRedirect(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	r := httptest.NewRequest("", "/assets/"+fn, nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "/assets/data.d10b4c3f" {
		t.Errorf("expected Location header %q, got %q", "/assets/data.d10b4c3f", loc)
	}
}

func TestServerHasherWithExtension(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets/data.d10b4c3f.txt", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	}).ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerHasherRedirectWithExtension(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	r := httptest.NewRequest("", "/assets/"+fn, nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "/assets/data.d10b4c3f.txt" {
		t.Errorf("expected Location header %q, got %q", "/assets/data.d10b4c3f.txt", loc)
	}
}

func TestServerHasherNoRegularFile(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, code)
	}
}

func TestServerHasherRedirectTrailingSlash(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets/data.d10b4c3f.txt/", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher:                MD5Hasher{8},
		RedirectTrailingSlash: true,
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "/assets/data.d10b4c3f.txt" {
		t.Errorf("expected Location header %q, got %q", "/assets/data.d10b4c3f.txt", loc)
	}
}

func TestServerHasherRedirectTrailingSlashCanonicalPath(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	r := httptest.NewRequest("", "/assets/data.txt/", nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher:                MD5Hasher{8},
		RedirectTrailingSlash: true,
	}).ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, code)
	}

	loc := w.Result().Header.Get("Location")
	if loc != "/assets/data.d10b4c3f.txt" {
		t.Errorf("expected Location header %q, got %q", "/assets/data.d10b4c3f.txt", loc)
	}
}

type faultyHasher struct{}

func (f faultyHasher) Hash(io.Reader) (string, error) {
	return "", errTest
}

func (f faultyHasher) IsHash(string) bool {
	return false
}

func TestServerInternalServerError(t *testing.T) {
	dir := t.TempDir()

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString("not to be hashed")
	if err != nil {
		t.Error(err)
	}
	f.Close()

	h := New("/assets", dir, &Options{
		Hasher: faultyHasher{},
	})

	r := httptest.NewRequest("", "/assets/data.txt", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	content := "Internal Server Error\n"
	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerInternalServerErrorCustom(t *testing.T) {
	dir := t.TempDir()

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString("not to be hashed")
	if err != nil {
		t.Error(err)
	}
	f.Close()

	content := "Test"
	h := New("/assets", dir, &Options{
		Hasher: faultyHasher{},
		InternalServerErrorHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, content, http.StatusTeapot)
		}),
	})

	r := httptest.NewRequest("", "/assets/data.txt", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusTeapot {
		t.Errorf("expected status code %d, got %d", http.StatusTeapot, code)
	}

	body := w.Body.String()
	if body != content+"\n" {
		t.Errorf("expected content %q, got %q", content+"\n", body)
	}
}

type nullHasher struct{}

func (f nullHasher) Hash(io.Reader) (string, error) {
	return "", nil
}

func (f nullHasher) IsHash(string) bool {
	return false
}

func TestServerHasherNullHasher(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	r := httptest.NewRequest("", "/assets/"+fn, nil)
	w := httptest.NewRecorder()

	New("/assets", dir, &Options{
		Hasher: nullHasher{},
	}).ServeHTTP(w, r)

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerHashedPath(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "data"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	p, err := New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	}).HashedPath(fn)

	if err != nil {
		t.Error(err)
	}

	want := "/assets/data.d10b4c3f"
	if p != want {
		t.Errorf("expected hashed path %q, got %q", want, p)
	}
}

func TestServerNoHasher(t *testing.T) {
	dir := t.TempDir()

	content := "file content"

	f, err := os.Create(filepath.Join(dir, "main.js"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	_, fn := filepath.Split(f.Name())
	f.Close()

	p, err := New("/assets", dir, nil).HashedPath(fn)

	if err != nil {
		t.Error(err)
	}

	want := "/assets/main.js"
	if p != want {
		t.Errorf("expected hashed path %q, got %q", want, p)
	}
}

func TestServerHashedPathError(t *testing.T) {
	dir := t.TempDir()

	f, err := os.Create(filepath.Join(dir, "data.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString("not to be hashed")
	if err != nil {
		t.Error(err)
	}
	f.Close()

	p, err := New("/assets", dir, &Options{
		Hasher: faultyHasher{},
	}).HashedPath("data.txt")

	if err != errTest {
		t.Errorf("expected error %v, got %v", errTest, err)
	}

	if p != "" {
		t.Errorf("expected path %q, got %q", "", p)
	}
}

func TestServerGetHashedPath(t *testing.T) {
	dir := t.TempDir()

	content := "gopher"
	f, err := os.Create(filepath.Join(dir, "data.12345678.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	h := New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	})

	r := httptest.NewRequest("", "/assets/data.12345678.txt", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	code := w.Result().StatusCode
	if code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, code)
	}

	body := w.Body.String()
	if body != content {
		t.Errorf("expected content %q, got %q", content, body)
	}
}

func TestServerHashedPathFromFilename(t *testing.T) {
	dir := t.TempDir()

	content := "gopher"
	f, err := os.Create(filepath.Join(dir, "data.12345678.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	h := New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
	})

	got, err := h.HashedPath("data.txt")
	if err != nil {
		t.Error(err)
	}
	expected := "/assets/data.12345678.txt"
	if got != expected {
		t.Errorf("expected hashed path %q, got %q", expected, got)
	}
}

func TestServerHashedPathFromFilenameWithAltDir(t *testing.T) {
	dir := t.TempDir()

	altDir := t.TempDir()

	content := "file alt content"

	if err := os.WriteFile(filepath.Join(altDir, "data.12345678.txt"), []byte(content), 0666); err != nil {
		t.Error(err)
	}

	h := New("/assets", dir, &Options{
		Hasher: MD5Hasher{8},
		AltDir: altDir,
	})

	got, err := h.HashedPath("data.txt")
	if err != nil {
		t.Error(err)
	}
	expected := "/assets/data.12345678.txt"
	if got != expected {
		t.Errorf("expected hashed path %q, got %q", expected, got)
	}
}

func TestServerHashedPathFromFilenameWithFilenames(t *testing.T) {
	dir := t.TempDir()

	content := "gopher"
	f, err := os.Create(filepath.Join(dir, "data.12345678.txt"))
	if err != nil {
		t.Error(err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		t.Error(err)
	}
	f.Close()

	h := New("/assets", dir, &Options{
		Hasher:    MD5Hasher{8},
		Filenames: []string{filepath.Join(dir, "data.abcdef01.txt")},
	})

	got, err := h.HashedPath("data.txt")
	if err != nil {
		t.Error(err)
	}
	expected := "/assets/data.abcdef01.txt"
	if got != expected {
		t.Errorf("expected hashed path %q, got %q", expected, got)
	}
}

func TestServerHashedPathFromFilenameWithAltDirWithFilenames(t *testing.T) {
	dir := t.TempDir()

	altDir := t.TempDir()

	content := "file alt content"

	if err := os.WriteFile(filepath.Join(altDir, "data.12345678.txt"), []byte(content), 0666); err != nil {
		t.Error(err)
	}

	h := New("/assets", dir, &Options{
		Hasher:    MD5Hasher{8},
		AltDir:    altDir,
		Filenames: []string{filepath.Join(altDir, "data.abcdef01.txt")},
	})

	got, err := h.HashedPath("data.txt")
	if err != nil {
		t.Error(err)
	}
	expected := "/assets/data.abcdef01.txt"
	if got != expected {
		t.Errorf("expected hashed path %q, got %q", expected, got)
	}
}
