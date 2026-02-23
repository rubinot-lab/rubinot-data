package api

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_http_requests_total",
			Help: "Total HTTP requests by route, method, and status code",
		},
		[]string{"route", "method", "status_code"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_http_request_duration_seconds",
			Help:    "Duration of HTTP requests by route and method",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"route", "method"},
	)
	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_http_response_size_bytes",
			Help:    "Response size in bytes by route and method",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000},
		},
		[]string{"route", "method"},
	)
	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rubinotdata_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)
	validationRejections = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_validation_rejections_total",
			Help: "Total validation rejections by endpoint and error code",
		},
		[]string{"endpoint", "error_code"},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		httpResponseSize,
		httpRequestsInFlight,
		validationRejections,
	)
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpRequestsInFlight.Inc()
		started := time.Now()

		c.Next()

		httpRequestsInFlight.Dec()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		duration := time.Since(started).Seconds()
		responseSize := float64(c.Writer.Size())

		httpRequestsTotal.WithLabelValues(route, method, statusCode).Inc()
		httpRequestDuration.WithLabelValues(route, method).Observe(duration)
		httpResponseSize.WithLabelValues(route, method).Observe(responseSize)
	}
}
