package metric

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func newHttpRequestStatusCounterMetric() *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "evsched",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests",
		},
		[]string{
			"method",
			"path",
			"status",
		},
	)
}

func newHttpDurationMetric() *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "evsched",
			Subsystem: "http",
			Name:      "duration_seconds",
			Help:      "HTTP request duration.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{
			"method",
			"path",
		},
	)
}

func (m *Metrics) ObserveHttpRequest(method string, path string, status int, duration time.Duration) {
	m.httpRequests.WithLabelValues(
		method,
		path,
		strconv.Itoa(status),
	).Inc()

	m.httpDuration.WithLabelValues(
		method, path).Observe(duration.Seconds())
}
