package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	Requests *prometheus.CounterVec
	Duration *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		Requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total HTTP requests",
		}, []string{"method", "path"}),

		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "request_duration_seconds",
				Buckets: []float64{0.05, 0.1, 0.5, 1, 2.5},
			},
			[]string{}),
	}
	reg.MustRegister(m.Requests)
	reg.MustRegister(m.Duration)
	return m
}
