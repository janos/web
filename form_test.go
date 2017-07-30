// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import "testing"

func TestFormError(t *testing.T) {
	e := FormErrors{}
	if e.HasErrors() {
		t.Error("expected false from HasErrors")
	}

	text := "test error"
	text2 := "test error 2"
	e = NewError(text)
	if !e.HasErrors() {
		t.Error("expected true from HasErrors")
	}
	if e.Errors[0] != text {
		t.Errorf("expected %q, got %q", text, e.Errors[0])
	}
	e.AddError(text2)
	if e.Errors[1] != text2 {
		t.Errorf("expected %q, got %q", text2, e.Errors[1])
	}
}

func TestFormError_Field(t *testing.T) {
	field := "field"
	text := "test error"
	field2 := "field 2"
	text2 := "test error 2"
	e := NewFieldError(field, text)
	if !e.HasErrors() {
		t.Error("expected true from HasErrors")
	}
	if e.FieldErrors[field][0] != text {
		t.Errorf("expected %q, got %q", text, e.FieldErrors[field][0])
	}
	e.AddFieldError(field2, text2)
	if e.FieldErrors[field2][0] != text2 {
		t.Errorf("expected %q, got %q", text2, e.FieldErrors[field2][0])
	}
}
