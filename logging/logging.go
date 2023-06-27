// Copyright (c) 2022, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"resenje.org/iostuff"
)

// NewApplicationLoggerCloser construct a logger and returns a closer of its
// writer. It uses ApplicationLogWriteCloser with
// iostuff.NewDailyReplaceableWriterConstructor for log rotation.
func NewApplicationLoggerCloser(dir, name string, newHandler func(io.Writer) slog.Handler, fallback io.Writer) (l *slog.Logger, closeFunc func() error) {
	w := ApplicationLogWriteCloser(dir, name, fallback)
	return slog.New(newHandler(w)), w.Close
}

// NewTextHandler calls slog.NewTextHandler but returns the Logger interface to
// be used as an argument in NewApplicationLoggerCloser.
func NewTextHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewTextHandler(w, opts)
}

// NewJSONHandler calls slog.NewJSONHandler but returns the Logger interface to
// be used as an argument in NewApplicationLoggerCloser.
func NewJSONHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewJSONHandler(w, opts)
}

// ApplicationLogWriteCloser returns a writer which is a daily rotated file
// constructed using NewDailyReplaceableWriterConstructor.
func ApplicationLogWriteCloser(dir, name string, fallback io.Writer) io.WriteCloser {
	if dir == "" {
		if wc, ok := fallback.(io.WriteCloser); ok {
			return wc
		}
		return iostuff.NewNopWriteCloser(fallback)
	}
	return iostuff.NewReplaceableWriter(NewDailyReplaceableWriterConstructor(dir, name))
}

// NewDailyReplaceableWriterConstructor creates a writer constructor that can be
// used for daily rotated files for iostuff.NewReplaceableWriter. The files are
// named using pattern dir/2006/01/02/name.log where dir and name are passed
// arguments and the date parameters are year, month and day values of the
// current time.
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
		r = r.WithContext(NewSlogContext(r.Context(), l))
		h.ServeHTTP(w, r)
	})
}

// HandlerKey is a log key for the handler name added by HandlerLogger function.
const HandlerKey = "handler"

// HandlerLogger provides a logger from HTTP request with attached name of the
// handler.
func HandlerLogger(r *http.Request, handlerName string) *slog.Logger {
	return SlogFromContext(r.Context()).With(HandlerKey, handlerName)
}

type contextKeySlog struct{}

// NewSlogContext returns a context that contains the given Logger.
// Use FromContext to retrieve the Logger.
func NewSlogContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKeySlog{}, l)
}

// SlogFromContext returns the Logger stored in ctx by NewContext, or the default
// Logger if there is none.
func SlogFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKeySlog{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
