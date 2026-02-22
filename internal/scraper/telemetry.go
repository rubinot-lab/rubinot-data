package scraper

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/giovannirco/rubinot-data/internal/scraper")

var (
	scrapeRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinot_scrape_requests_total",
			Help: "Total scrape requests by endpoint and status",
		},
		[]string{"endpoint", "status"},
	)
	scrapeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinot_scrape_duration_seconds",
			Help:    "Duration of scrape requests by endpoint",
			Buckets: []float64{0.25, 0.5, 1, 2, 3, 5, 8, 13, 21},
		},
		[]string{"endpoint"},
	)
	parseDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinot_parse_duration_seconds",
			Help:    "Duration of parsing operations by endpoint",
			Buckets: []float64{0.01, 0.03, 0.06, 0.1, 0.2, 0.5, 1, 2},
		},
		[]string{"endpoint"},
	)
)

func init() {
	prometheus.MustRegister(scrapeRequests, scrapeDuration, parseDuration)
}
