module github.com/giovannirco/rubinot-data

go 1.23

require (
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/gin-gonic/gin v1.11.0
	github.com/go-resty/resty/v2 v2.16.5
	github.com/prometheus/client_golang v1.23.2
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.63.0
	go.opentelemetry.io/otel v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.38.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.38.0
	go.opentelemetry.io/otel/sdk v1.38.0
)
