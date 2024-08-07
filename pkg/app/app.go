package app

import (
	"context"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/vmkteam/brokersrv/pkg/rpcqueue"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
)

type Config struct {
	Server struct {
		Host string
		Port int
	}
	NATS struct {
		URL string
	}
	Settings struct {
		RpcServices []string
	}
}

type App struct {
	appName string
	cfg     Config
	echo    *echo.Echo
	nc      *nats.Conn
	js      jetstream.JetStream

	qm *QueueManager
}

func New(appName string, cfg Config, nc *nats.Conn) *App {
	a := &App{
		appName: appName,
		cfg:     cfg,
		echo:    echo.New(),
	}
	a.echo.HideBanner = true
	a.echo.HidePort = true
	a.nc = nc

	return a
}

// Run is a function that runs application.
func (a *App) Run() error {
	a.registerDebugHandlers()
	a.registerHandlers()
	a.registerMetrics()
	if err := a.registerJetStream(); err != nil {
		return err
	}
	a.qm = NewQueueManager(a.js)

	return a.runHTTPServer(a.cfg.Server.Host, a.cfg.Server.Port)
}

// registerJetStream configure and register stream for NATS JetStream
func (a *App) registerJetStream() error {
	ctx := context.Background()
	js, err := jetstream.New(a.nc)
	if err != nil {
		return err
	}
	a.js = js

	jsCfg := jetstream.StreamConfig{
		Name:      rpcqueue.StreamName,
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Subjects:  []string{rpcqueue.StreamName + ".*"},
	}

	_, err = js.CreateOrUpdateStream(ctx, jsCfg)

	return err
}

// Shutdown is a function that gracefully stops HTTP server.
func (a *App) Shutdown(timeout time.Duration) {
	if err := a.nc.Drain(); err != nil {
		log.Printf("shutting down NATS connection err=%q", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.echo.Shutdown(ctx); err != nil {
		log.Printf("shutting down server err=%q", err)
	}
}
