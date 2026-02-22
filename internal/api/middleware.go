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
			Help:    "Duration of HTTP requests by route, method, and status code",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"route", "method", "status_code"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
}

func metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unknown"
		}

		statusCode := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		duration := time.Since(started).Seconds()

		httpRequestsTotal.WithLabelValues(route, method, statusCode).Inc()
		httpRequestDuration.WithLabelValues(route, method, statusCode).Observe(duration)
	}
}
