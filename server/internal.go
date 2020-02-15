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

func newInternalRouter(s *Server) http.Handler {
	//
	// Top level internal router
	//
	internalBaseRouter := http.NewServeMux()

	//
	// Internal router
	//
	internalRouter := http.NewServeMux()
	internalBaseRouter.Handle("/", web.ChainHandlers(
		handlers.CompressHandler,
		s.textRecoveryHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(internalRouter),
	))
	internalRouter.Handle("/", http.HandlerFunc(textNotFoundHandler))
	internalRouter.Handle("/status", http.HandlerFunc(s.statusHandler))
	internalRouter.Handle("/data", datadump.Handler(s.dataDumpServices, s.name+"_"+s.Version(), s.logger))

	internalRouter.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	internalRouter.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	internalRouter.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	internalRouter.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	internalRouter.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	internalRouter.Handle("/debug/vars", expvar.Handler())

	//
	// Internal API router
	//
	internalAPIRouter := http.NewServeMux()
	internalBaseRouter.Handle("/api/", web.ChainHandlers(
		handlers.CompressHandler,
		s.jsonRecoveryHandler,
		web.NoCacheHeadersHandler,
		web.FinalHandler(internalAPIRouter),
	))
	internalAPIRouter.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jsonhttp.NotFound(w, nil)
	}))
	internalAPIRouter.Handle("/api/status", http.HandlerFunc(s.statusAPIHandler))
	if s.maintenanceService != nil {
		internalAPIRouter.Handle("/api/maintenance", jsonMethodHandler{
			"GET":    http.HandlerFunc(s.maintenanceService.StatusHandler),
			"POST":   http.HandlerFunc(s.maintenanceService.OnHandler),
			"DELETE": http.HandlerFunc(s.maintenanceService.OffHandler),
		})
	}
	internalBaseRouter.Handle("/metrics", promhttp.InstrumentMetricHandler(
		s.metricsRegistry,
		promhttp.HandlerFor(s.metricsRegistry, promhttp.HandlerOpts{}),
	))

	//
	// Final internal handler
	//
	return web.ChainHandlers(
		func(h http.Handler) http.Handler {
			return web.NewSetHeadersHandler(h, map[string]string{
				"Server": s.name + "/" + s.Version(),
			})
		},
		web.FinalHandler(internalBaseRouter),
	)
}
