// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package server is an extremely opinionated package for gluing together HTTP
// servers, managing their listeners, TLS certificates, domains, metrics and
// data dumps.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/acme/autocert"
	"resenje.org/email"
	"resenje.org/logging"
	"resenje.org/recovery"
	"resenje.org/web/maintenance"
	"resenje.org/web/servers"
	httpServer "resenje.org/web/servers/http"
	"resenje.org/x/datadump"
)

// Default values for HTTP server timeouts.
var (
	DefaultIdleTimeout  = 30 * time.Minute
	DefaultReadTimeout  = 1 * time.Minute
	DefaultWriteTimeout = 1 * time.Minute
)

// Server contains all required properties, services and functions
// to provide core functionality.
type Server struct {
	name           string
	version        string
	buildInfo      string
	acmeCertsDir   string
	acmeCertsEmail string
	logger         *logging.Logger

	dataDumpServices   map[string]datadump.Interface
	emailService       *email.Service
	recoveryService    *recovery.Service
	maintenanceService *maintenance.Service

	startTime       time.Time
	servers         *servers.Servers
	metricsRegistry *prometheus.Registry
}

// New initializes new server with provided options.
func New(o Options) (s *Server, err error) {
	if o.Name == "" {
		o.Name = "server"
	}
	if o.Version == "" {
		o.Version = "0"
	}
	s = &Server{
		name:               o.Name,
		version:            o.Version,
		buildInfo:          o.BuildInfo,
		acmeCertsDir:       o.ACMECertsDir,
		acmeCertsEmail:     o.ACMECertsEmail,
		logger:             o.Logger,
		dataDumpServices:   make(map[string]datadump.Interface),
		emailService:       o.EmailService,
		recoveryService:    o.RecoveryService,
		maintenanceService: o.MaintenanceService,
		startTime:          time.Now().Round(0),
		servers: servers.New(
			servers.WithLogger(o.Logger),
			servers.WithRecoverFunc(o.RecoveryService.Recover),
		),
		metricsRegistry: prometheus.NewRegistry(),
	}

	// register standard metrics
	s.metricsRegistry.MustRegister(
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		prometheus.NewGoCollector(),
	)

	var certificates []tls.Certificate
	if o.InternalTLSKey != "" && o.InternalTLSCert != "" {
		cert, err := tls.LoadX509KeyPair(o.InternalTLSCert, o.InternalTLSKey)
		if err != nil {
			return nil, fmt.Errorf("load certificate: %v", err)
		}
		certificates = append(certificates, cert)
	}
	tlsConfig := &tls.Config{
		Certificates:       certificates,
		MinVersion:         tls.VersionTLS10,
		NextProtos:         []string{"h2"},
		ClientSessionCache: tls.NewLRUClientSessionCache(-1),
	}

	internalRouter := newInternalRouter(s, o.SetupInternalRouters)
	if o.ListenInternal != "" {
		s.servers.Add("internal HTTP", o.ListenInternal, httpServer.New(
			internalRouter,
		))
	}
	if o.ListenInternalTLS != "" {
		s.servers.Add("internal TLS HTTP", o.ListenInternalTLS, httpServer.New(
			internalRouter,
			httpServer.WithTLSConfig(tlsConfig),
		))
	}
	return s, nil
}

// Options structure contains optional properties for the Server.
type Options struct {
	Name                 string
	Version              string
	BuildInfo            string
	ListenInternal       string
	ListenInternalTLS    string
	InternalTLSCert      string
	InternalTLSKey       string
	ACMECertsDir         string
	ACMECertsEmail       string
	SetupInternalRouters func(base, api *http.ServeMux)

	Logger *logging.Logger

	EmailService       *email.Service
	RecoveryService    *recovery.Service
	MaintenanceService *maintenance.Service
}

// HTTPOptions holds parameters for WithHTTP method.
// If timeouts are not set, package Default values
// will be used.
type HTTPOptions struct {
	Handlers     Handlers
	Name         string
	Listen       string
	ListenTLS    string
	TLSCerts     []TLSCert
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// SetHandler sets an HTTP handler to serve specific domains.
func (o *HTTPOptions) SetHandler(h http.Handler, domains ...string) {
	if o.Handlers == nil {
		o.Handlers = NewHandlers()
	}
	o.Handlers.Set(h, domains...)
}

// Handlers maps HTTP handlers to domains.
type Handlers map[string]http.Handler

// NewHandlers constructs new instance of Handlers.
func NewHandlers() (h Handlers) {
	return make(Handlers)
}

// Set sets an HTTP handler to serve specific domains.
// If domain list is empty, this handler will be used
// as Default one.
func (dh Handlers) Set(h http.Handler, domains ...string) Handlers {
	if domains == nil {
		dh[""] = h
		return dh
	}
	for _, domain := range domains {
		dh[domain] = h
	}
	return dh
}

// TLSCert holds filesystem paths to TLS certificate and its key.
type TLSCert struct {
	Cert string
	Key  string
}

// WithHTTP add an HTTP server with a specific name, that listens for plain http
// or encrypted connections to the list of servers.
func (s *Server) WithHTTP(o HTTPOptions) (err error) {
	_, httpsPort, _ := net.SplitHostPort(o.ListenTLS)
	handlers := make(map[string]http.Handler)
	DefaultHandler, ok := o.Handlers[""]
	if !ok {
		DefaultHandler = http.HandlerFunc(textNotFoundHandler)
	}
	for domain, handler := range o.Handlers {
		if domain == "" {
			continue
		}
		handlers[domain] = handler
	}

	for domain := range handlers {
		var redirectDomain string
		if strings.HasPrefix(domain, "www.") {
			redirectDomain = strings.TrimPrefix(domain, "www.")
		} else {
			redirectDomain = "www." + domain
		}
		if _, ok := handlers[redirectDomain]; !ok {
			handlers[redirectDomain] = newRedirectDomainHandler(domain, httpsPort)
		}
	}

	var router http.Handler
	if len(handlers) > 0 {
		router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = r.Host
			}
			h, ok := handlers[host]
			if ok {
				h.ServeHTTP(w, r)
				return
			}
			DefaultHandler.ServeHTTP(w, r)
		})
	} else {
		router = DefaultHandler
	}

	var certificates []tls.Certificate
	for _, c := range o.TLSCerts {
		if c.Cert != "" && c.Key != "" {
			cert, err := tls.LoadX509KeyPair(c.Cert, c.Key)
			if err != nil {
				return fmt.Errorf("load certificate: %v", err)
			}
			certificates = append(certificates, cert)
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       certificates,
		MinVersion:         tls.VersionTLS10,
		NextProtos:         []string{"h2"},
		ClientSessionCache: tls.NewLRUClientSessionCache(-1),
	}
	var acmeHTTPHandler func(fallback http.Handler) http.Handler
	if s.acmeCertsDir != "" && o.ListenTLS != "" {
		certManager := autocert.Manager{
			Prompt: autocert.AcceptTOS,
			Cache:  autocert.DirCache(s.acmeCertsDir),
		}
		domains := make([]string, 0, len(handlers))
		for d := range handlers {
			domains = append(domains, d)
		}
		certManager.HostPolicy = autocert.HostWhitelist(domains...)
		certManager.Email = s.acmeCertsEmail

		tlsConfig = certManager.TLSConfig()
		tlsConfig.MinVersion = tls.VersionTLS10
		tlsConfig.ClientSessionCache = tls.NewLRUClientSessionCache(-1)
		tlsConfig.Certificates = certificates
		acmeHTTPHandler = certManager.HTTPHandler
	}

	idleTimeout := o.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = DefaultIdleTimeout
	}
	readTimeout := o.ReadTimeout
	if readTimeout == 0 {
		readTimeout = DefaultReadTimeout
	}
	writeTimeout := o.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = DefaultWriteTimeout
	}

	if o.Listen != "" {
		h := router
		if acmeHTTPHandler != nil {
			h = acmeHTTPHandler(h)
		}
		if httpsPort != "" {
			h = redirectHTTPSHandler(h, httpsPort)
		}
		server := httpServer.New(h)
		server.IdleTimeout = idleTimeout
		server.ReadTimeout = readTimeout
		server.WriteTimeout = writeTimeout
		n := "HTTP"
		if o.Name != "" {
			n = o.Name + " HTTP"
		}
		s.servers.Add(n, o.Listen, server)
	}

	if o.ListenTLS != "" {
		server := httpServer.New(
			router,
			httpServer.WithTLSConfig(tlsConfig),
		)
		server.IdleTimeout = idleTimeout
		server.ReadTimeout = readTimeout
		server.WriteTimeout = writeTimeout
		n := "HTTPS"
		if o.Name != "" {
			n = o.Name + " HTTPS"
		}
		s.servers.Add(n, o.ListenTLS, server)
	}

	return nil
}

// Serve starts servers.
func (s *Server) Serve() error {
	return s.servers.Serve()
}

// Shutdown gracefully terminates servers.
func (s *Server) Shutdown(ctx context.Context) {
	s.servers.Shutdown(ctx)
}

// WithMetrics registers prometheus collector to be exposed
// on internal handler /metrics request.
func (s *Server) WithMetrics(cs ...prometheus.Collector) {
	s.metricsRegistry.MustRegister(cs...)
}

// WithDataDumpService adds a service with data dump interface
// to be used on internal handler /data request.
func (s *Server) WithDataDumpService(name string, service datadump.Interface) {
	s.dataDumpServices[name] = service
}

// Version returns server version with build info data
// suffixed if exists.
func (s *Server) Version() (v string) {
	if s.buildInfo != "" {
		return fmt.Sprintf("%s-%s", s.version, s.buildInfo)
	}
	return s.version
}
