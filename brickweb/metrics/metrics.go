/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func init() {
	prometheus.MustRegister(httpInflightGauge, httpRequestCounter, httpRequestDuration, httpResponseSize, httpRequestSize, DepcrecationWarnings)
	prometheus.MustRegister(HttpChallengeAttempts, ChallengeHttpRequests, ChallengeAttempts)
}

var (
	httpInflightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_inflight_requests",
		Help: "A gauge of requests currently being served by the wrapped Handler",
	})
	httpRequestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "A counter of total requests served",
	},
		[]string{"code", "method"})
	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "A Histogram for request latencies",
		Buckets: []float64{.25, .5, 1, 2.5, 5, 10},
	},
		[]string{"code"})
	httpResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_response_size_bytes",
		Help:    "A Histogram for Response sizes",
		Buckets: []float64{200, 500, 900, 1500},
	},
		[]string{})
	httpRequestSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_size_bytes",
		Help:    "A Histogram for Request sizes",
		Buckets: []float64{200, 500, 900, 1500, 5000},
	},
		[]string{"code"})
	DepcrecationWarnings = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "api_deprecation_warnings_count",
		Help: "A Counter for all Deprecation Warnings; mainly requests compliant wiht older RFC-Drafts, but not with the current Version",
	}, []string{"endpoint", "type"})
	HttpChallengeAttempts = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "va_http01_challenges_total",
		Help: "A Counter for total started http-01 challenge attempts",
	})
	ChallengeHttpRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "va_http_requests_total",
		Help: "A Counter for all Challenge attempts total",
	}, []string{"code"})
	ChallengeAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "va_challenge_attempts_total",
		Help: "A counter for all challenge attempts total",
	}, []string{"success", "method"})
)

//FullInstrumentingHandler instruments the passed handler to observe all relevant and observable http metrics
var FullInstrumentingHandler = fullInstrumentingHandler

//FulLInstrumentingHanlderFunc takes a http.HandlerFunc instead, satisfying the signature of vestigo.Middleware
var FullInstrumentingHandlerFunc = func(h http.HandlerFunc) http.HandlerFunc {
	return FullInstrumentingHandler(h)
}

func fullInstrumentingHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		promhttp.InstrumentHandlerInFlight(httpInflightGauge,
			promhttp.InstrumentHandlerDuration(httpRequestDuration,
				promhttp.InstrumentHandlerCounter(httpRequestCounter,
					promhttp.InstrumentHandlerResponseSize(httpResponseSize,
						promhttp.InstrumentHandlerRequestSize(httpRequestSize, h))))).ServeHTTP(w, r)
	}
}
