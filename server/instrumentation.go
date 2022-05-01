// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package server

import (
	"expvar"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"resenje.org/jsonhttp"
	"resenje.org/x/datadump"

	"resenje.org/web"
)

func newInstrumentationRouter(s *Server, setupFunc func(base, api *http.ServeMux)) http.Handler {
	//
	// Top level instrumentation router
	//
	baseRouter := http.NewServeMux()

	//
	// Instrumentation router
	//
	instrumentationRouter := http.NewServeMux()
	baseRouter.Handle("/", web.ChainHandlers(
		handlers.CompressHandler,
		s.textRecoveryHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(instrumentationRouter),
	))
	instrumentationRouter.Handle("/", http.HandlerFunc(textNotFoundHandler))
	instrumentationRouter.Handle("/status", http.HandlerFunc(s.statusHandler))
	instrumentationRouter.Handle("/data", datadump.Handler(s.dataDumpServices, s.name+"_"+s.Version(), s.logger))

	instrumentationRouter.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	instrumentationRouter.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	instrumentationRouter.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	instrumentationRouter.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	instrumentationRouter.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	instrumentationRouter.Handle("/debug/vars", expvar.Handler())

	//
	// Instrumentation API router
	//
	instrumentationAPIRouter := http.NewServeMux()
	baseRouter.Handle("/api/", web.ChainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(instrumentationAPIRouter),
	))
	instrumentationAPIRouter.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonhttp.NotFound(w, nil)
	}))
	instrumentationAPIRouter.Handle("/api/status", http.HandlerFunc(s.statusAPIHandler))
	if s.maintenanceService != nil {
		instrumentationAPIRouter.Handle("/api/maintenance", jsonMethodHandler{
			"GET":    http.HandlerFunc(s.maintenanceService.StatusHandler),
			"POST":   http.HandlerFunc(s.maintenanceService.OnHandler),
			"DELETE": http.HandlerFunc(s.maintenanceService.OffHandler),
		})
	}
	baseRouter.Handle("/metrics", promhttp.InstrumentMetricHandler(
		s.metricsRegistry,
		promhttp.HandlerFor(s.metricsRegistry, promhttp.HandlerOpts{}),
	))

	if setupFunc != nil {
		setupFunc(baseRouter, instrumentationAPIRouter)
	}

	//
	// Final instrumentation handler
	//
	return web.ChainHandlers(
		func(h http.Handler) http.Handler {
			return web.NewSetHeadersHandler(h, map[string]string{
				"Server": s.name + "/" + s.Version(),
			})
		},
		web.FinalHandler(baseRouter),
	)
}
