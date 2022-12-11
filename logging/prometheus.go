// Copyright (c) 2022 Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package logging

import (
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
)

// Metrics holds Prometheus counters.
type Metrics struct {
	vector *prometheus.CounterVec
}

// MetricsOptions holds options for NewCounter constructor.
type MetricsOptions struct {
	Namespace string
	Subsystem string
	Name      string
	Help      string
}

// NewMetrics creates new Counter instance. Options value can be nil.
func NewMetrics(options *MetricsOptions) (c *Metrics) {
	if options == nil {
		options = new(MetricsOptions)
	}
	if options.Subsystem == "" {
		options.Subsystem = "logger"
	}
	if options.Name == "" {
		options.Name = "messages_total"
	}
	if options.Help == "" {
		options.Help = "Number of log messages processed, partitioned by log level."
	}
	vector := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: options.Namespace,
			Subsystem: options.Subsystem,
			Name:      options.Name,
			Help:      options.Help,
		},
		[]string{"level"},
	)
	return &Metrics{
		vector: vector,
	}
}

func (c *Metrics) Inc(level slog.Level) {
	c.vector.WithLabelValues(level.String()).Inc()
}

// Metrics returns all Prometheus metrics that should be registered.
func (c *Metrics) Metrics() (cs []prometheus.Collector) {
	return []prometheus.Collector{c.vector}
}
