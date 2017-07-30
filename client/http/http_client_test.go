// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpClient

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	r, err := Default.Get(ts.URL)
	if err != nil {
		t.Error(err)
	}
	if r == nil {
		t.Error("unexpected nil response")
	}
}

func TestClientRetry(t *testing.T) {
	l, err := net.Listen("tcp", "")
	if err != nil {
		t.Error(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	go func() {
		time.Sleep(2 * time.Second)

		l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Error(err)
		}
		defer l.Close()
		server := http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		}
		if err := server.Serve(l); err != nil {
			t.Error(err)
		}
	}()

	r, err := New(&Options{RetryTimeMax: 10 * time.Second}).Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		t.Error(err)
	}
	if r == nil {
		t.Error("unexpected nil response")
	}

	r, err = (&http.Client{Transport: Transport(&Options{RetryTimeMax: 10 * time.Second})}).Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		t.Error(err)
	}
	if r == nil {
		t.Error("unexpected nil response")
	}
}

func TestClientRetryFailure(t *testing.T) {
	l, err := net.Listen("tcp", "")
	if err != nil {
		t.Error(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	go func() {
		time.Sleep(2 * time.Second)

		l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			t.Error(err)
		}
		defer l.Close()
		server := http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		}
		if err := server.Serve(l); err != nil {
			t.Error(err)
		}
	}()

	r, err := New(&Options{RetryTimeMax: 1 * time.Second}).Get(fmt.Sprintf("http://localhost:%d", port))
	if err == nil || !strings.Contains(err.Error(), "getsockopt: connection refused") {
		t.Errorf("expected connection refused error, got %#v", err)
	}
	if r != nil {
		t.Error("unexpected not-nil response")
	}

	r, err = (&http.Client{Transport: Transport(&Options{RetryTimeMax: 1 * time.Second})}).Get(fmt.Sprintf("http://localhost:%d", port))
	if err == nil || !strings.Contains(err.Error(), "getsockopt: connection refused") {
		t.Errorf("expected connection refused error, got %#v", err)
	}
	if r != nil {
		t.Error("unexpected not-nil response")
	}
}

func TestClientRedirectHeaders(t *testing.T) {
	want := "test web"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		got := r.Header.Get("X-Test")
		if got != want {
			t.Errorf("expected X-Test header %q, got %q", want, got)
		}
	}))
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/redirect", nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("X-Test", want)
	r, err := Default.Do(req)
	if err != nil {
		t.Error(err)
	}
	if r == nil {
		t.Error("unexpected nil response")
	}
}

func TestOptionsMarshalJSON(t *testing.T) {
	o := Options{
		Timeout:             30 * time.Second,
		KeepAlive:           3 * time.Minute,
		TLSHandshakeTimeout: 100 * time.Millisecond,
		TLSSkipVerify:       true,
		RetryTimeMax:        1 * time.Hour,
		RetrySleepMax:       10 * time.Second,
		RetrySleepBase:      10 * time.Microsecond,
	}

	b, err := o.MarshalJSON()
	if err != nil {
		t.Error(err.Error())
	}
	got := string(b)
	want := "{\"timeout\":\"30s\",\"keep-alive\":\"3m0s\",\"tls-handshake-timeout\":\"100ms\",\"tls-skip-verify\":true,\"retry-time-max\":\"1h0m0s\",\"retry-sleep-max\":\"10s\",\"retry-sleep-base\":\"10µs\"}"
	if got != want {
		t.Errorf("expected json %q, got %q", want, got)
	}
}

func TestOptionsUnmarshalJSON(t *testing.T) {
	o := &Options{}
	if err := o.UnmarshalJSON([]byte("{\"timeout\":\"30s\",\"keep-alive\":\"3m0s\",\"tls-handshake-timeout\":\"100ms\",\"tls-skip-verify\":true,\"retry-time-max\":\"1h0m0s\",\"retry-sleep-max\":\"10s\",\"retry-sleep-base\":\"10µs\"}")); err != nil {
		t.Error(err.Error())
	}

	Timeout := 30 * time.Second
	if o.Timeout != Timeout {
		t.Errorf("expected Timeout %v, got %v", Timeout, o.Timeout)
	}
	KeepAlive := 3 * time.Minute
	if o.KeepAlive != KeepAlive {
		t.Errorf("expected KeepAlive %v, got %v", KeepAlive, o.KeepAlive)
	}
	TLSHandshakeTimeout := 100 * time.Millisecond
	if o.TLSHandshakeTimeout != TLSHandshakeTimeout {
		t.Errorf("expected TLSHandshakeTimeout %v, got %v", TLSHandshakeTimeout, o.TLSHandshakeTimeout)
	}
	TLSSkipVerify := true
	if o.TLSSkipVerify != TLSSkipVerify {
		t.Errorf("expected TLSSkipVerify %v, got %v", TLSSkipVerify, o.TLSSkipVerify)
	}
	RetryTimeMax := 1 * time.Hour
	if o.RetryTimeMax != RetryTimeMax {
		t.Errorf("expected RetryTimeMax %v, got %v", RetryTimeMax, o.RetryTimeMax)
	}
	RetrySleepMax := 10 * time.Second
	if o.RetrySleepMax != RetrySleepMax {
		t.Errorf("expected RetrySleepMax %v, got %v", RetrySleepMax, o.RetrySleepMax)
	}
	RetrySleepBase := 10 * time.Microsecond
	if o.RetrySleepBase != RetrySleepBase {
		t.Errorf("expected RetrySleepBase %v, got %v", RetrySleepBase, o.RetrySleepBase)
	}
}

func TestOptionsUnmarshalJSONError(t *testing.T) {
	o := &Options{}
	e := "invalid character '1' looking for beginning of object key string"
	if err := o.UnmarshalJSON([]byte("{1}")); err == nil || err.Error() != e {
		t.Errorf("expected error %q, got %v", e, err)
	}
}
