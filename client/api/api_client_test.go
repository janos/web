// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type TestResponseJSON struct {
	Method  string
	URL     *url.URL
	Headers map[string][]string
	Body    string
}

func TestClient(t *testing.T) {
	contentType := "application/json; charset=utf-8"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/empty":
		case "/test.html":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`Test`))
		case "/syntax-error.json":
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write([]byte(`{1}`))
		case "/type-error.json":
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write([]byte(`{"method":123}`))
		case "/bad-request":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`Bad data...`))
		case "/internal-server-error":
			w.WriteHeader(http.StatusInternalServerError)
		case "/entity-not-found":
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"entity not found"}`))
		case "/error-invalid-message-type":
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":1}`))
		case "/json-syntax-error":
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{1}`))
		case "/error-1000":
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":1000}`))
		case "/2s-request":
			time.Sleep(2 * time.Second)
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"code":200}`))
		default:
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			r.Body.Close()
			b, err := json.Marshal(TestResponseJSON{
				Method:  r.Method,
				URL:     r.URL,
				Headers: r.Header,
				Body:    string(body),
			})
			if err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(b)
		}
	}))
	defer ts.Close()

	t.Run("Basic", func(t *testing.T) {
		client := New(ts.URL, nil)
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		if r == nil {
			t.Error("response must not be nil")
		}
		if err := client.JSON("GET", "/", nil, nil, nil); err != nil {
			t.Error(err.Error())
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("EndpointWithoutScheme", func(t *testing.T) {
		client := New(ts.URL[7:], nil)
		if _, err := client.Request("GET", "/", nil, nil, nil); err != nil {
			t.Error(err.Error())
		}
		if err := client.JSON("GET", "/", nil, nil, nil); err != nil {
			t.Error(err.Error())
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("EndpointInvalidEndpoint", func(t *testing.T) {
		client := New("", nil)
		want := "Get http:///: http: no Host in request URL"
		if _, err := client.Request("GET", "/", nil, nil, nil); err.Error() != want {
			t.Errorf("expected error %q, got %v", want, err)
		}
		if err := client.JSON("GET", "/", nil, nil, nil); err.Error() != want {
			t.Errorf("expected error %q, got %v", want, err)
		}
		if _, _, err := client.Stream("GET", "/", nil, nil, nil); err.Error() != want {
			t.Errorf("expected error %q, got %v", want, err)
		}
	})

	t.Run("DefaultHTTPClient", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.HTTPClient = nil
		if _, err := client.Request("GET", "/", nil, nil, nil); err != nil {
			t.Error(err.Error())
		}
		if err := client.JSON("GET", "/", nil, nil, nil); err != nil {
			t.Error(err.Error())
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("UserAgent", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.UserAgent = "Testing 1.1"
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.Headers["User-Agent"][0] != client.UserAgent {
			t.Errorf("expected user agent %q, got %q", client.UserAgent, want.Headers["User-Agent"][0])
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Testing 1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("CustomHeaders", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.Headers = map[string]string{}
		client.Headers["One"] = "1"
		client.Headers["Two"] = "2"
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.Headers["One"][0] != "1" {
			t.Errorf("expected header One %q, got %q", "1", want.Headers["One"][0])
		}
		if want.Headers["Two"][0] != "2" {
			t.Errorf("expected header Two %q, got %q", "2", want.Headers["Two"][0])
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"One":["1"],"Two":["2"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("Accept", func(t *testing.T) {
		client := New(ts.URL, nil)
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept":["application/json","application/xml"],"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, []string{"application/json", "application/xml"})
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, []string{"application/json", "application/xml"})
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("Key", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.Key = "KeyValue"
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.Headers[DefaultKeyHeader][0] != client.Key {
			t.Errorf("expected header %q %q, got %q", DefaultKeyHeader, client.Key, want.Headers[DefaultKeyHeader][0])
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"],"X-Key":["KeyValue"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("KeyCustomHeader", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.Key = "KeyCustomValue"
		client.KeyHeader = "X-Custom-Key"
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.Headers[client.KeyHeader][0] != client.Key {
			t.Errorf("expected header %q %q, got %q", client.KeyHeader, client.Key, want.Headers[client.KeyHeader][0])
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"],"X-Custom-Key":["KeyCustomValue"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("BasicAuth", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.BasicAuth = &BasicAuth{
			Username: "test-username",
			Password: "test-password",
		}
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		wantHeader := "Basic dGVzdC11c2VybmFtZTp0ZXN0LXBhc3N3b3Jk"
		if want.Headers["Authorization"][0] != wantHeader {
			t.Errorf("expected Authorization header %q, got %q", wantHeader, want.Headers["Authorization"][0])
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"Authorization":["Basic dGVzdC11c2VybmFtZTp0ZXN0LXBhc3N3b3Jk"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("KeyEmptyHeader", func(t *testing.T) {
		client := New(ts.URL, nil)
		client.Key = "KeyValue"
		client.KeyHeader = ""
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.Headers[DefaultKeyHeader][0] != client.Key {
			t.Errorf("expected header %q %q, got %q", DefaultKeyHeader, client.Key, want.Headers[DefaultKeyHeader][0])
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"],"X-Key":["KeyValue"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("Path", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := &TestResponseJSON{}
		path := "/test-path"
		if err := client.JSON("GET", path, nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.URL.Path != path {
			t.Errorf("expected url path %q, got %q", path, want.URL.Path)
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("PathWithEndpointPath", func(t *testing.T) {
		client := New(ts.URL+"/subpath", nil)
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "/test-path", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		path := "/subpath/test-path"
		if want.URL.Path != path {
			t.Errorf("expected url path %q, got %q", path, want.URL.Path)
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/subpath/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("PathWithoutLeadingSlash", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := &TestResponseJSON{}
		if err := client.JSON("GET", "test-path", nil, nil, want); err != nil {
			t.Error(err.Error())
		}
		path := "/test-path"
		if want.URL.Path != path {
			t.Errorf("expected url path %q, got %q", path, want.URL.Path)
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("Query", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := &TestResponseJSON{}
		query := url.Values{}
		query.Set("limit", "100")
		if err := client.JSON("GET", "/", query, nil, want); err != nil {
			t.Error(err.Error())
		}
		if want.URL.Query().Get("limit") != "100" {
			t.Errorf("expected url path %q, got %q", "100", want.URL.Query().Get("limit"))
		}
		body := `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("Body", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := &TestResponseJSON{}
		body := "Test Body"
		if err := client.JSON("GET", "/", nil, strings.NewReader(body), want); err != nil {
			t.Error(err.Error())
		}
		if want.Body != body {
			t.Errorf("expected url path %q, got %q", body, want.Body)
		}
		body = `{"Method":"GET","URL":{"Scheme":"","Opaque":"","User":null,"Host":"","Path":"/","RawPath":"","ForceQuery":false,"RawQuery":"","Fragment":""},"Headers":{"Accept-Encoding":["gzip"],"User-Agent":["Go-http-client/1.1"]},"Body":""}`
		r, err := client.Request("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err.Error())
		}
		r.Body.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		data, ct, err := client.Stream("GET", "/", nil, nil, nil)
		if err != nil {
			t.Error(err.Error())
		}
		b, err = ioutil.ReadAll(data)
		if err != nil {
			t.Error(err.Error())
		}
		data.Close()
		if string(b) != body {
			t.Errorf("expected body %q, got %q", body, string(b))
		}
		if ct != contentType {
			t.Errorf("expected Content-Type header %q, got %q", contentType, ct)
		}
	})

	t.Run("JSONEmptyResponseBody", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "empty response body"
		if err := client.JSON("GET", "/empty", nil, nil, &TestResponseJSON{}); err == nil || err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err)
		}
	})

	t.Run("JSONUnsupportedContentType", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "unsupported content type: text/html"
		if err := client.JSON("GET", "/test.html", nil, nil, &TestResponseJSON{}); err == nil || err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err)
		}
	})

	t.Run("JSONSyntaxError", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "json invalid character '1' looking for beginning of object key string, line: 1, column: 2"
		if err := client.JSON("GET", "/syntax-error.json", nil, nil, &TestResponseJSON{}); err == nil || err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err)
		}
	})

	t.Run("JSONTypeError", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "expected json string value but got number, line: 1, column: 13"
		if err := client.JSON("GET", "/type-error.json", nil, nil, &TestResponseJSON{}); err == nil || err.Error() != want {
			t.Errorf("expected error %q, got %q", want, err)
		}
	})

	t.Run("Error400", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "http status: bad request"
		code := 400
		_, err := client.Request("GET", "/bad-request", nil, nil, nil)
		errLocal, ok := err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		err = client.JSON("GET", "/bad-request", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		_, _, err = client.Stream("GET", "/bad-request", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
	})

	t.Run("Error500", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "http status: internal server error"
		code := 500
		_, err := client.Request("GET", "/internal-server-error", nil, nil, nil)
		errLocal, ok := err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		err = client.JSON("GET", "/internal-server-error", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		_, _, err = client.Stream("GET", "/internal-server-error", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
	})

	t.Run("ErrorWithCustomMessage", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "entity not found"
		code := 404
		_, err := client.Request("GET", "/entity-not-found", nil, nil, nil)
		errLocal, ok := err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		err = client.JSON("GET", "/entity-not-found", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		_, _, err = client.Stream("GET", "/entity-not-found", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
	})

	t.Run("ErrorWithInvalidMessageResponseType", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "http status: not found"
		code := 404
		_, err := client.Request("GET", "/error-invalid-message-type", nil, nil, nil)
		errLocal, ok := err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		err = client.JSON("GET", "/error-invalid-message-type", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		_, _, err = client.Stream("GET", "/error-invalid-message-type", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
	})

	t.Run("ErrorWithJSONSyntaxError", func(t *testing.T) {
		client := New(ts.URL, nil)
		want := "json invalid character '1' looking for beginning of object key string, line: 1, column: 2"
		code := 404
		_, err := client.Request("GET", "/json-syntax-error", nil, nil, nil)
		errLocal, ok := err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		err = client.JSON("GET", "/json-syntax-error", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
		_, _, err = client.Stream("GET", "/json-syntax-error", nil, nil, nil)
		errLocal, ok = err.(*Error)
		if !ok {
			t.Errorf("error %#v is not Error", err)
		} else {
			if errLocal.Error() != want {
				t.Errorf("expected error %q, got %v", want, err)
			}
			if errLocal.Code != code {
				t.Errorf("expected error code %d, got %d", code, errLocal.Code)
			}
		}
	})

	t.Run("ErrorFromErrorRegistry", func(t *testing.T) {
		registry := NewMapErrorRegistry(nil, nil)
		code := 1000
		if err := registry.AddError(code, errTest); err != nil {
			t.Error(err.Error())
		}
		client := New(ts.URL, registry)
		if _, err := client.Request("GET", "/error-1000", nil, nil, nil); err != errTest {
			t.Errorf("expected error %v, got %v", errTest, err)
		}
		if err := client.JSON("GET", "/error-1000", nil, nil, nil); err != errTest {
			t.Errorf("expected error %v, got %v", errTest, err)
		}
		if _, _, err := client.Stream("GET", "/error-1000", nil, nil, nil); err != errTest {
			t.Errorf("expected error %v, got %v", errTest, err)
		}
	})

	t.Run("ErrorFromErrorRegistryHandler", func(t *testing.T) {
		registry := NewMapErrorRegistry(nil, nil)
		code := 1000
		if err := registry.AddHandler(code, errHandler); err != nil {
			t.Error(err.Error())
		}
		client := New(ts.URL, registry)
		if _, err := client.Request("GET", "/error-1000", nil, nil, nil); err != errHandlerTest {
			t.Errorf("expected error %v, got %v", errHandlerTest, err)
		}
		if err := client.JSON("GET", "/error-1000", nil, nil, nil); err != errHandlerTest {
			t.Errorf("expected error %v, got %v", errHandlerTest, err)
		}
		if _, _, err := client.Stream("GET", "/error-1000", nil, nil, nil); err != errHandlerTest {
			t.Errorf("expected error %v, got %v", errHandlerTest, err)
		}
	})

	t.Run("Context", func(t *testing.T) {
		client := New(ts.URL, nil)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := client.RequestContext(ctx, "GET", "/2s-request", nil, nil, nil)
		if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
		}
		err = client.JSONContext(ctx, "GET", "/2s-request", nil, nil, nil)
		if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
		}
		_, _, err = client.StreamContext(ctx, "GET", "/2s-request", nil, nil, nil)
		if !strings.Contains(err.Error(), context.DeadlineExceeded.Error()) {
			t.Errorf("expected error %v, got %v", context.DeadlineExceeded, err)
		}
	})
}
