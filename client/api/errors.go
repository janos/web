// Copyright (c) 2016, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apiClient

// Error represents a HTTP error that contains status text and status code.
type Error struct {
	// HTTP response status text.
	Status string
	// HTTP response status code.
	Code int
}

func (e *Error) Error() string {
	return e.Status
}
