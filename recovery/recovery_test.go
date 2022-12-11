// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recovery

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/exp/slog"
)

var (
	panicMessage = "HTTP utils panic!"
	panicHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		panic(panicMessage)
	})
	req = func() *http.Request {
		req, _ := http.NewRequest("GET", "/", nil)
		return req
	}()
)

func TestHandler(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	New(panicHandler).ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), panicMessage) {
		t.Errorf("got %q, expected %q", buf.String(), panicMessage)
	}
}

func TestHandlerLabel(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	label := "test-handler-name 0.1"

	New(panicHandler, WithLabel(label)).ServeHTTP(httptest.NewRecorder(), req)

	if !strings.Contains(buf.String(), label) {
		t.Errorf("got %q, expected %q", buf.String(), label)
	}
}

func TestHandlerResponse(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	responseBody := "Test recovery handler!"
	recovery := New(panicHandler, WithPanicResponse(responseBody, ""))
	recorder := httptest.NewRecorder()
	recovery.ServeHTTP(recorder, req)

	if !strings.Contains(recorder.Body.String(), responseBody) {
		t.Errorf("got %q, expected %q", recorder.Body.String(), responseBody)
	}
}

func TestHandlerPanicResponseHandler(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	recovery := New(panicHandler, WithPanicResponseHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))
	recorder := httptest.NewRecorder()
	recovery.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTeapot {
		t.Errorf("got %d, expected %d", recorder.Code, http.StatusTeapot)
	}
}

func TestHandlerLogger(t *testing.T) {
	var buf bytes.Buffer

	New(panicHandler, WithLogger(slog.New(slog.NewTextHandler(&buf)))).ServeHTTP(httptest.NewRecorder(), req)

	want := "level=ERROR msg=\"http recovery handler\" method=GET url=/ err=\"HTTP utils panic!\" debug="
	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, expected %q", buf.String(), want)
	}
}

func TestHandlerNotifier(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	subject := ""
	body := ""
	done := make(chan struct{})
	notifyFunc := func(s, b string) error {
		subject = s
		body = b
		close(done)
		return nil
	}

	New(panicHandler, WithNotifier(NotifierFunc(notifyFunc))).ServeHTTP(httptest.NewRecorder(), req)

	<-done

	if !strings.Contains(subject, "Panic GET /:") {
		t.Errorf("got %q, expected %q", subject, "Panic GET /:")
	}
	if !strings.Contains(body, "runtime/debug.Stack") {
		t.Errorf("got %q, expected %q", body, "runtime/debug.Stack")
	}
}
