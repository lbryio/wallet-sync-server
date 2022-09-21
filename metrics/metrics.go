package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_sync_requests_count",

			// TODO For some reason, help text nor type seem to show up in
			// Prometheus?
			Help: "Total number of requests to various endpoints",
		},
		[]string{"method", "endpoint"},
	)
	ErrorsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_sync_error_count",
			Help: "Total number of various kinds of errors",
		},
		[]string{"error_type"},
	)
)

func init() {
	prometheus.MustRegister(RequestsCount)
	prometheus.MustRegister(ErrorsCount)
}
