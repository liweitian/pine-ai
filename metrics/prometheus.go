package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	inferRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pineai_infer_requests_total",
			Help: "Total infer requests by model/version.",
		},
		[]string{"model", "version"},
	)

	inferFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pineai_infer_failures_total",
			Help: "Total failed infer requests by model/version.",
		},
		[]string{"model", "version"},
	)

	inferDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pineai_infer_duration_seconds",
			Help:    "Infer duration in seconds by model/version.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"model", "version"},
	)
)

func init() {
	prometheus.MustRegister(inferRequestsTotal)
	prometheus.MustRegister(inferFailuresTotal)
	prometheus.MustRegister(inferDurationSeconds)
}

func ObserveInfer(model, version string, success bool, duration time.Duration) {
	inferRequestsTotal.WithLabelValues(model, version).Inc()
	if !success {
		inferFailuresTotal.WithLabelValues(model, version).Inc()
	}
	inferDurationSeconds.WithLabelValues(model, version).Observe(duration.Seconds())
}
