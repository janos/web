// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

import "testing"

func TestError(t *testing.T) {
	want := "http test error"
	got := (&Error{Status: want, Code: 1000}).Error()
	if want != got {
		t.Errorf("expected %q, got %q", want, got)
	}
}
