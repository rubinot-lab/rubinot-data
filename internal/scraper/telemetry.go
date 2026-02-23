package scraper

import (
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/giovannirco/rubinot-data/internal/scraper")

var (
	scrapeRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_scrape_requests_total",
			Help: "Total scrape requests by endpoint and status",
		},
		[]string{"endpoint", "status"},
	)
	scrapeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_scrape_duration_seconds",
			Help:    "Duration of scrape requests by endpoint",
			Buckets: []float64{0.25, 0.5, 1, 2, 3, 5, 8, 13, 21},
		},
		[]string{"endpoint"},
	)
	parseDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_parse_duration_seconds",
			Help:    "Duration of parsing operations by endpoint",
			Buckets: []float64{0.01, 0.03, 0.06, 0.1, 0.2, 0.5, 1, 2},
		},
		[]string{"endpoint"},
	)

	FlareSolverrRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_flaresolverr_requests_total",
			Help: "Total FlareSolverr requests by status",
		},
		[]string{"status"},
	)
	FlareSolverrDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_flaresolverr_duration_seconds",
			Help:    "Duration of FlareSolverr HTTP calls",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60, 120},
		},
	)
	CloudflareChallenges = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rubinotdata_cloudflare_challenges_total",
			Help: "Total Cloudflare challenge pages detected",
		},
	)

	UpstreamStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_upstream_status_total",
			Help: "Upstream HTTP status codes returned via FlareSolverr",
		},
		[]string{"endpoint", "status_code"},
	)
	UpstreamMaintenance = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rubinotdata_upstream_maintenance_total",
			Help: "Total upstream maintenance mode detections",
		},
	)

	ParseErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_parse_errors_total",
			Help: "Total parser errors by endpoint and error type",
		},
		[]string{"endpoint", "error_type"},
	)
	ParseItems = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubinotdata_parse_items_total",
			Help: "Number of items returned by last parse per endpoint",
		},
		[]string{"endpoint"},
	)

	WorldsDiscovered = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rubinotdata_worlds_discovered",
			Help: "Number of worlds discovered at startup",
		},
	)
	WorldPlayersOnline = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubinotdata_world_players_online",
			Help: "Players online per world",
		},
		[]string{"world"},
	)
	WorldsTotalPlayersOnline = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rubinotdata_worlds_total_players_online",
			Help: "Total players online across all worlds",
		},
	)
	ValidatorRefresh = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_validator_refresh_total",
			Help: "Validator refresh attempts by status",
		},
		[]string{"status"},
	)
	ValidatorRefreshDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_validator_refresh_duration_seconds",
			Help:    "Duration of validator refresh operations",
			Buckets: []float64{0.5, 1, 2, 5, 10, 20, 30, 60},
		},
	)

	CacheRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_cache_requests_total",
			Help: "Cache requests by endpoint and result (placeholder)",
		},
		[]string{"endpoint", "result"},
	)
	CacheDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rubinotdata_cache_duration_seconds",
			Help:    "Cache operation duration by endpoint (placeholder)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		},
		[]string{"endpoint"},
	)
	CacheEntries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rubinotdata_cache_entries",
			Help: "Number of cached entries per endpoint (placeholder)",
		},
		[]string{"endpoint"},
	)
	CacheStaleServes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rubinotdata_cache_stale_serves_total",
			Help: "Total stale cache serves per endpoint (placeholder)",
		},
		[]string{"endpoint"},
	)
)

func init() {
	prometheus.MustRegister(
		scrapeRequests,
		scrapeDuration,
		parseDuration,
		FlareSolverrRequests,
		FlareSolverrDuration,
		CloudflareChallenges,
		UpstreamStatus,
		UpstreamMaintenance,
		ParseErrors,
		ParseItems,
		WorldsDiscovered,
		WorldPlayersOnline,
		WorldsTotalPlayersOnline,
		ValidatorRefresh,
		ValidatorRefreshDuration,
		CacheRequests,
		CacheDuration,
		CacheEntries,
		CacheStaleServes,
	)
}
