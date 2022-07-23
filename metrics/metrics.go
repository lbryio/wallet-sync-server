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
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(RequestsCount)
}
