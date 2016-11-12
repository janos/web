// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fileServer

import (
	"errors"
	"strings"
	"testing"
)

func TestMD5HasherHash(t *testing.T) {
	h, err := MD5Hasher{7}.Hash(strings.NewReader("test"))
	if err != nil {
		t.Error(err)
	}
	want := "098f6bc"
	if want != h {
		t.Errorf("expected hash %q, got %q", want, h)
	}
}

func TestMD5HasherHashLength(t *testing.T) {
	h, err := MD5Hasher{100}.Hash(strings.NewReader("test"))
	if err != nil {
		t.Error(err)
	}
	want := ""
	if want != h {
		t.Errorf("expected hash %q, got %q", want, h)
	}
}

var errTest = errors.New("test error")

type faultyReader struct{}

func (f faultyReader) Read(p []byte) (n int, err error) {
	err = errTest
	return
}

func TestMD5HasherHashError(t *testing.T) {
	h, err := MD5Hasher{100}.Hash(faultyReader{})
	if err != errTest {
		t.Errorf("expected error %v, got %v", errTest, err)
	}
	want := ""
	if want != h {
		t.Errorf("expected hash %q, got %q", want, h)
	}
}

func TestMD5HasherIsHash(t *testing.T) {
	is := MD5Hasher{9}.IsHash("123abcdef")
	if !is {
		t.Error("hash \"123abcdef\" not reported that it is a valid hash of length 9")
	}
}

func TestMD5HasherIsHashFalse(t *testing.T) {
	is := MD5Hasher{9}.IsHash("123abcdeg")
	if is {
		t.Error("hash \"123abcdeg\" reported that it is a valid hash of length 9")
	}
}

func TestMD5HasherIsHashLength(t *testing.T) {
	is := MD5Hasher{5}.IsHash("123")
	if is {
		t.Error("hash \"123\" reported that it is a valid hahs of length 5")
	}
}
