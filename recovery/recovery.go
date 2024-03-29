// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package recovery

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Handler implements http.Handler interface that will recover from panic
// and return appropriate HTTP response, log and notify on such event.
type Handler struct {
	handler              http.Handler
	label                string
	panicBody            string
	panicContentType     string
	panicResponseHandler http.Handler
	logger               *slog.Logger
	notifier             Notifier
}

// Option is a function that sets optional parameters to the Handler.
type Option func(*Handler)

// WithLabel sets a string that will be included in log message and
// notification. Usually, it contains the name of the server
// and its version.
func WithLabel(label string) Option { return func(o *Handler) { o.label = label } }

// WithPanicResponse sets a fixed body and its content type HTTP header
// that will be returned as HTTP response on panic event.
// If WithPanicResponseHandler is defined, this options are ignored.
func WithPanicResponse(body, contentType string) Option {
	return func(o *Handler) {
		o.panicBody = body
		o.panicContentType = contentType
	}
}

// WithPanicResponseHandler sets http.Handler that will be executed on
// panic event. It is useful when the response has dynamic content.
// If the content is static it is better to use WithPanicResponse option
// instead. This option has a precedence upon WithPanicResponse.
func WithPanicResponseHandler(h http.Handler) Option {
	return func(o *Handler) { o.panicResponseHandler = h }
}

// WithLogger sets the function that will perform message logging.
// Default is slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(o *Handler) { o.logger = l }
}

// WithNotifier sets the function that takes subject and body
// arguments and is intended for sending notifications.
func WithNotifier(notifier Notifier) Option { return func(o *Handler) { o.notifier = notifier } }

// New creates a new Handler from the handler that is wrapped and
// protected with recover function.
func New(handler http.Handler, options ...Option) (h *Handler) {
	h = &Handler{
		handler: handler,
		logger:  slog.Default(),
	}
	for _, option := range options {
		option(h)
	}
	return
}

// ServeHTTP implements http.Handler interface.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer func() {
		if err := recover(); err != nil {
			debugMsg := fmt.Sprintf(
				"%s\n\n%#v\n\n%#v",
				debug.Stack(),
				r.URL,
				r.Header,
			)
			if h.label != "" {
				debugMsg = h.label + "\n\n" + debugMsg
			}
			h.logger.ErrorContext(ctx, "http recovery handler", "method", r.Method, "url", r.URL.String(), "error", err, "debug", debugMsg)

			if h.notifier != nil {
				go func() {
					defer func() {
						if err := recover(); err != nil {
							h.logger.ErrorContext(ctx, "http recovery handler: notify panic", slog.Any("error", err))
						}
					}()

					if err := h.notifier.Notify(
						fmt.Sprint(
							"Panic ",
							r.Method,
							" ",
							r.URL.String(),
							": ", err,
						),
						debugMsg,
					); err != nil {
						h.logger.ErrorContext(ctx, "http recovery handler: notify", slog.Any("error", err))
					}
				}()
			}

			if h.panicResponseHandler != nil {
				h.panicResponseHandler.ServeHTTP(w, r)
				return
			}

			if h.panicContentType != "" {
				w.Header().Set("Content-Type", h.panicContentType)
			}
			w.WriteHeader(http.StatusInternalServerError)
			if h.panicBody != "" {
				fmt.Fprintln(w, h.panicBody)
			}
		}
	}()

	h.handler.ServeHTTP(w, r)
}

// Notifier defines functionalities required for sending notifications.
type Notifier interface {
	Notify(subject, body string) error
}

// NotifierFunc type is an adapter to allow the use of
// ordinary functions as Notifier.
type NotifierFunc func(subject, body string) error

// Notify calls NotifierFunc(subject, body).
func (f NotifierFunc) Notify(subject, body string) error {
	return f(subject, body)
}
