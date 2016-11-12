// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import (
	"errors"
	"testing"
)

var errTest = errors.New("test error")

func TestMapErrorRegistry(t *testing.T) {
	r := NewMapErrorRegistry(nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err.Error())
	}
	if err := r.Error(code); err != errTest {
		t.Errorf("expected error %v, got %v", errTest, err)
	}
}

func TestMapErrorRegistryErrorAlreadyRegistered(t *testing.T) {
	r := NewMapErrorRegistry(nil)
	code := 1000
	if err := r.AddError(code, errTest); err != nil {
		t.Error(err.Error())
	}
	if err := r.AddError(code, errTest); err != ErrErrorAlreadyRegistered {
		t.Errorf("expected error %v, got %v", ErrErrorAlreadyRegistered, err)
	}
}
