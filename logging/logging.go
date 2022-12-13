// Copyright (c) 2022, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/exp/slog"
	"resenje.org/iostuff"
)

func ApplicationLogWriteCloser(dir, name string, fallback io.Writer) io.WriteCloser {
	if dir == "" {
		if wc, ok := fallback.(io.WriteCloser); ok {
			return wc
		}
		return iostuff.NewNopWriteCloser(fallback)
	}
	return iostuff.NewReplaceableWriter(NewDailyReplaceableWriterConstructor(dir, name))
}

func NewDailyReplaceableWriterConstructor(dir, name string) func(flag string) (f io.Writer, today string, err error) {
	return func(flag string) (io.Writer, string, error) {
		today := time.Now().Format("2006/01/02")
		if today == flag {
			return nil, "", nil
		}
		filename := filepath.Join(dir, today) + "/" + name + ".log"
		if err := os.MkdirAll(filepath.Dir(filename), 0o777); err != nil {
			return nil, "", fmt.Errorf("create log directory: %v", err)
		}
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o666)
		if err != nil {
			return nil, "", fmt.Errorf("open log file: %v", err)
		}
		return f, today, nil
	}
}

// NewContextLoggerHandler injects logger into HTTP request Context.
// HandlerLogger function can be used to get the logger and attach a handler
// name.
func NewContextLoggerHandler(h http.Handler, l *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(slog.NewContext(r.Context(), l))
		h.ServeHTTP(w, r)
	})
}

const HandlerKey = "handler"

// HandlerLogger provides a logger from HTTP request with attached name of the
// handler.
func HandlerLogger(r *http.Request, handlerName string) *slog.Logger {
	return slog.FromContext(r.Context()).With(HandlerKey, handlerName)
}
