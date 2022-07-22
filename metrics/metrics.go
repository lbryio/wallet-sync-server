package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "requests_count",
			Help: "Total number of requests to various endpoints",
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(RequestsCount)
}
