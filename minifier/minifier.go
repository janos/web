// Copyright (c) 2018, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package minifier

import (
	"io"
	"net/http"
	"regexp"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

var minifier = minify.New()

func init() {
	htmlMinify := &html.Minifier{
		KeepWhitespace: true,
	}
	minifier.AddFunc("text/css", css.Minify)
	minifier.Add("text/html", htmlMinify)
	minifier.Add("text/x-template", htmlMinify)
	minifier.AddFunc("text/javascript", js.Minify)
	minifier.AddFunc("application/javascript", js.Minify)
	minifier.AddFunc("application/x-javascript", js.Minify)
	minifier.AddFunc("image/svg+xml", svg.Minify)
	minifier.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	minifier.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)
}

// LogFunc defines the function that is used by NewHandler
// to log error messages from minifier.
type LogFunc func(format string, a ...interface{})

// NewHandler returns a minifer http handler that implements common http.Hander
// related interfaces and uses a common configuration for github.com/tdewolff/minify.
func NewHandler(h http.Handler, logFunc LogFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mw := &minifyResponseWriter{mw: minifier.ResponseWriter(w, r), w: w}
		h.ServeHTTP(mw, r)
		if err := mw.Close(); err != nil && err != minify.ErrNotExist && logFunc != nil {
			logFunc("minifier %q: %v", r.URL.String(), err)
		}
	})
}

type minifyResponseWriter struct {
	mw http.ResponseWriter
	w  http.ResponseWriter
}

func (w *minifyResponseWriter) Header() http.Header {
	return w.mw.Header()
}

func (w *minifyResponseWriter) CloseNotify() <-chan bool {
	return w.w.(http.CloseNotifier).CloseNotify()
}

func (w *minifyResponseWriter) Flush() {
	w.w.(http.Flusher).Flush()
}

func (w *minifyResponseWriter) Write(b []byte) (int, error) {
	return w.mw.Write(b)
}

func (w *minifyResponseWriter) WriteHeader(s int) {
	w.mw.WriteHeader(s)
}

func (w *minifyResponseWriter) Push(target string, opts *http.PushOptions) error {
	return w.w.(http.Pusher).Push(target, opts)
}

func (w *minifyResponseWriter) Close() error {
	if c, ok := w.mw.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
