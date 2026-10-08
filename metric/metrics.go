package metric

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry     *prometheus.Registry
	httpRequests *prometheus.CounterVec
	httpDuration *prometheus.HistogramVec
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	m := &Metrics{
		registry:     reg,
		httpRequests: newHttpRequestStatusCounterMetric(),
		httpDuration: newHttpDurationMetric(),
	}

	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.httpRequests,
		m.httpDuration,
	)

	return m
}

type HttpHandler struct {
	http.Handler
}

func (m *Metrics) NewHttpHandler() *HttpHandler {
	return &HttpHandler{Handler: promhttp.HandlerFor(
		m.registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)}
}
