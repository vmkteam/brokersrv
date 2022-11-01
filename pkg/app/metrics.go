package app

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// registerMetrics is a function that initializes a.stat* variables and adds /metrics endpoint to echo.
func (a *App) registerMetrics() {
	a.echo.Use(HTTPMetrics(a.appName))
	a.echo.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
}

//nolint:unused
func (a *App) newCounterMetric(subsystem, name, help string, labels ...string) *prometheus.CounterVec {
	metric := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: a.appName,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, labels)
	prometheus.MustRegister(metric)
	return metric
}

//nolint:unused
func (a *App) newGaugeMetric(subsystem, name, help string, labels ...string) *prometheus.GaugeVec {
	metric := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: a.appName,
		Subsystem: subsystem,
		Name:      name,
		Help:      help,
	}, labels)
	prometheus.MustRegister(metric)
	return metric
}
