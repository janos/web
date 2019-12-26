// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import (
	"errors"
	"testing"
)

var (
	errTest        = errors.New("test error")
	errHandlerTest = errors.New("test handler error")
)

func errHandler(body []byte) error {
	return errHandlerTest
}

func TestMapErrorRegistry(t *testing.T) {
	r := NewMapErrorRegistry(nil, nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err)
	}
	if err := r.Error(code); err != errTest {
		t.Errorf("expected error %v, got %v", errTest, err)
	}
	code = 1001
	errTestMessage, err := r.AddMessageError(code, "error message")
	if err != nil {
		t.Error(err)
	}
	if err := r.Error(code); err != errTestMessage {
		t.Errorf("expected error %v, got %v", errTestMessage, err)
	}
	code = 1002
	errTestMessage2 := r.MustAddMessageError(code, "error message2")
	if err := r.Error(code); err != errTestMessage2 {
		t.Errorf("expected error %v, got %v", errTestMessage2, err)
	}
	code = 1003
	if err := r.AddHandler(code, errHandler); err != nil {
		t.Error(err)
	}
	if handler := r.Handler(code); handler != nil {
		err := handler(nil)
		if err != errHandlerTest {
			t.Errorf("expected error %v, got %v", errHandlerTest, err)
		}
	}
}

func TestMapErrorRegistryErrorAlreadyRegistered(t *testing.T) {
	r := NewMapErrorRegistry(nil, nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err)
	}
	if err := r.AddError(code, errTest); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
	if _, err := r.AddMessageError(code, "message"); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
	if err := r.AddHandler(code, errHandler); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
	code = 2000
	if err := r.AddHandler(code, errHandler); err != nil {
		t.Error(err)
	}
	if err := r.AddHandler(code, errHandler); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
	if err := r.AddError(code, errTest); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
	if _, err := r.AddMessageError(code, "message"); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
}

func TestMapErrorRegistryMustAddErrorPanic(t *testing.T) {
	r := NewMapErrorRegistry(nil, nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err)
	}
	defer func() {
		if err := recover(); err != ErrErrorAlreadyRegistered {
			t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
		}
	}()
	r.MustAddError(code, errTest)
}

func TestMapErrorRegistryMustAddMessageErrorPanic(t *testing.T) {
	r := NewMapErrorRegistry(nil, nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err)
	}
	defer func() {
		if err := recover(); err != ErrErrorAlreadyRegistered {
			t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
		}
	}()
	if err := r.MustAddMessageError(code, "message"); err != nil {
		t.Error(err)
	}
}

func TestMapErrorRegistryMustAddHandlerPanic(t *testing.T) {
	r := NewMapErrorRegistry(nil, nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err)
	}
	defer func() {
		if err := recover(); err != ErrErrorAlreadyRegistered {
			t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
		}
	}()
	r.MustAddHandler(code, errHandler)
}
