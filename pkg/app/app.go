package app

import (
	"context"
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/stan.go"
)

type Config struct {
	Server struct {
		Host string
		Port int
	}
	NATS struct {
		URL       string
		ClusterID string
		ClientID  string
	}
	Settings struct {
		RpcServices []string
	}
}

type App struct {
	appName string
	cfg     Config
	echo    *echo.Echo

	qm *QueueManager
}

func New(appName string, cfg Config, sc stan.Conn) *App {
	a := &App{
		appName: appName,
		cfg:     cfg,
		echo:    echo.New(),
	}
	a.echo.HideBanner = true
	a.echo.HidePort = true

	a.qm = NewQueueManager(sc)

	return a
}

// Run is a function that runs application.
func (a *App) Run() error {
	a.registerDebugHandlers()
	a.registerHandlers()
	a.registerMetrics()
	return a.runHTTPServer(a.cfg.Server.Host, a.cfg.Server.Port)
}

// Shutdown is a function that gracefully stops HTTP server.
func (a *App) Shutdown(timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.echo.Shutdown(ctx); err != nil {
		log.Printf("shutting down server err=%q", err)
	}
}
