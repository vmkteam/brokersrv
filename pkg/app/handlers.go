package app

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/vmkteam/zenrpc/v2"
)

// runHTTPServer is a function that starts http listener using labstack/echo.
func (a *App) runHTTPServer(host string, port int) error {
	listenAddress := fmt.Sprintf("%s:%d", host, port)
	log.Printf("starting http listener at http://%s\n", listenAddress)

	return a.echo.Start(listenAddress)
}

// registerDebugHandlers adds /debug/pprof handlers into a.echo instance.
func (a *App) registerDebugHandlers() {
	dbg := a.echo.Group("/debug")

	// add pprof integration
	dbg.Any("/pprof/*", func(c echo.Context) error {
		if h, p := http.DefaultServeMux.Handler(c.Request()); p != "" {
			h.ServeHTTP(c.Response(), c.Request())
			return nil
		}
		return echo.NewHTTPError(http.StatusNotFound)
	})
}

func (a *App) registerHandlers() {
	a.echo.Any("/rpc/:service/", a.processRpcServices)
}

func (a *App) processRpcServices(c echo.Context) error {
	service := c.Param("service")

	if !a.serviceExists(service) {
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

	err = a.qm.Publish(c.Request().Context(), service, req, c.Request().Header)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, zenrpc.NewResponseError(nil, zenrpc.InternalError, err.Error(), nil))
	}

	return c.JSON(http.StatusOK, nil)
}

func (a *App) serviceExists(service string) bool {
	serviceExists := false
	for _, s := range a.cfg.Settings.RpcServices {
		if s == service {
			serviceExists = true
		}
	}
	return serviceExists
}
