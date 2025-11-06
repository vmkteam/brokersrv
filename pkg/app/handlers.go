package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vmkteam/brokersrv/pkg/rpcqueue"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vmkteam/appkit"
	"github.com/vmkteam/zenrpc/v2"
)

// runHTTPServer is a function that starts http listener using labstack/echo.
func (a *App) runHTTPServer(ctx context.Context, host string, port int) error {
	listenAddress := fmt.Sprintf("%s:%d", host, port)
	addr := "http://" + listenAddress
	a.Print(ctx, "starting http listener", "url", addr)

	// print registered services
	for _, srv := range a.cfg.Settings.RPCServices {
		a.Print(ctx, "register", "rpc", srv, "url", addr+"/"+srv+"/")
	}

	return a.echo.Start(listenAddress)
}

// registerDebugHandlers adds /debug/pprof handlers into a.echo instance.
func (a *App) registerDebugHandlers() {
	dbg := a.echo.Group("/debug")

	// add pprof integration
	dbg.Any("/pprof/*", appkit.PprofHandler)

	// add healthcheck
	a.echo.GET("/status", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// show all routes in devel mode
	if a.cfg.Server.IsDevel {
		a.echo.GET("/", appkit.RenderRoutes(a.appName, a.echo))
	}
}

// registerMetrics is a function that initializes a.stat* variables and adds /metrics endpoint to echo.
func (a *App) registerMetrics() {
	a.echo.Use(appkit.HTTPMetrics(appkit.DefaultServerName))
	a.echo.Any("/metrics", echo.WrapHandler(promhttp.Handler()))
}

func (a *App) registerHandlers() {
	a.echo.Any("/rpc/:service/", a.processRPCServices)
}

func (a *App) processRPCServices(c echo.Context) error {
	service := c.Param("service")
	stream := a.serviceStream(service)

	if stream == "" {
		return c.JSON(http.StatusInternalServerError, zenrpc.NewResponseError(nil, zenrpc.InvalidRequest, "service not exists", nil))
	}

	var req zenrpc.Request
	err := json.NewDecoder(c.Request().Body).Decode(&req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, zenrpc.NewResponseError(nil, zenrpc.ParseError, err.Error(), nil))
	}
	if req.ID != nil {
		return c.JSON(http.StatusInternalServerError, zenrpc.NewResponseError(nil, zenrpc.InvalidParams, "request ID not empty", nil))
	}

	err = a.qm.Publish(c.Request().Context(), stream, service, req, c.Request().Header)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, zenrpc.NewResponseError(nil, zenrpc.InternalError, err.Error(), nil))
	}

	return c.JSON(http.StatusOK, nil)
}

func (a *App) serviceStream(service string) string {
	for _, s := range a.cfg.Settings.RPCServices {
		if s == service {
			return rpcqueue.StreamName
		}
	}
	for _, s := range a.cfg.LegacySettings.RPCServices {
		if s == service {
			return rpcqueue.LegacyStreamName
		}
	}
	return ""
}

func (a *App) registerMiddlewares() {
	a.echo.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:    true,
		LogURI:       true,
		LogError:     true,
		HandleError:  true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogRequestID: true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.String("ip", v.RemoteIP),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.String("userAgent", v.UserAgent),
				slog.String("duration", v.Latency.String()),
				slog.String("xRequestId", v.RequestID),
				slog.String("platform", c.Request().Header.Get("Platform")),
				slog.String("version", c.Request().Header.Get("Version")),
			}

			if v.Error == nil {
				a.Log().LogAttrs(context.Background(), slog.LevelInfo, "http request", attrs...)
			} else {
				a.Log().LogAttrs(context.Background(), slog.LevelError, "http request error", append(attrs, slog.String("err", v.Error.Error()))...)
			}
			return nil
		},
	}))
}
