package api

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for ndc-loader.
var Metrics = struct {
	ProductsTotal   prometheus.Gauge
	PackagesTotal   prometheus.Gauge
	LoadDuration    prometheus.Gauge
	LoadLastSuccess prometheus.Gauge
	LoadErrorsTotal prometheus.Counter
	QueryDuration   *prometheus.HistogramVec
	SearchDuration  *prometheus.HistogramVec
}{
	ProductsTotal: promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ndc_loader_products_total",
		Help: "Current number of products in the database",
	}),
	PackagesTotal: promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ndc_loader_packages_total",
		Help: "Current number of packages in the database",
	}),
	LoadDuration: promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ndc_loader_load_duration_seconds",
		Help: "Duration of the last data load in seconds",
	}),
	LoadLastSuccess: promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ndc_loader_load_last_success_timestamp",
		Help: "Unix timestamp of the last successful data load",
	}),
	LoadErrorsTotal: promauto.NewCounter(prometheus.CounterOpts{
		Name: "ndc_loader_load_errors_total",
		Help: "Total number of failed load attempts",
	}),
	QueryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ndc_loader_query_duration_seconds",
		Help:    "Histogram of query latencies",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint"}),
	SearchDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ndc_loader_search_duration_seconds",
		Help:    "Histogram of search latencies",
		Buckets: prometheus.DefBuckets,
	}, []string{"endpoint"}),
}
