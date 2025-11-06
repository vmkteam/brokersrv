package app

import (
	"context"
	"fmt"
	"time"

	"github.com/vmkteam/brokersrv/pkg/rpcqueue"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/vmkteam/appkit"
	"github.com/vmkteam/embedlog"
)

//nolint:staticcheck
type Config struct {
	Server struct {
		Host    string
		Port    int
		IsDevel bool
	}
	NATS struct {
		URL            string
		StreamReplicas int
	}
	LegacySettings struct {
		RPCServices []string
	}
	Settings struct {
		RPCServices []string
	}
	Sentry struct {
		Environment string
		DSN         string
	}
}

type App struct {
	embedlog.Logger
	appName string
	cfg     Config
	echo    *echo.Echo
	nc      *nats.Conn
	js      jetstream.JetStream
	stream  jetstream.Stream

	qm *QueueManager
}

func New(appName string, sl embedlog.Logger, cfg Config, nc *nats.Conn) *App {
	a := &App{
		Logger:  sl,
		appName: appName,
		cfg:     cfg,
		echo:    appkit.NewEcho(),
	}
	a.nc = nc

	return a
}

// Run is a function that runs application.
func (a *App) Run(ctx context.Context) error {
	a.registerDebugHandlers()
	a.registerHandlers()
	a.registerMetrics()
	a.registerMiddlewares()

	if err := a.registerJetStream(ctx); err != nil {
		return err
	}
	a.qm = NewQueueManager(a.js)

	return a.runHTTPServer(ctx, a.cfg.Server.Host, a.cfg.Server.Port)
}

// registerJetStream configure and register stream for NATS JetStream
func (a *App) registerJetStream(ctx context.Context) error {
	var err error
	a.js, err = jetstream.New(a.nc)
	if err != nil {
		return fmt.Errorf("failed to get jetstream context: %w", err)
	}

	jsCfg := jetstream.StreamConfig{
		Name:      rpcqueue.StreamName,
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Subjects:  []string{rpcqueue.StreamName + ".*"},
		Replicas:  a.cfg.NATS.StreamReplicas,
	}

	a.stream, err = a.js.CreateOrUpdateStream(ctx, jsCfg)
	return err
}

// Shutdown is a function that gracefully stops HTTP server.
func (a *App) Shutdown(timeout time.Duration) error {
	if err := a.nc.Drain(); err != nil {
		return fmt.Errorf("NATS connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := a.nc.Drain(); err != nil {
		a.Print(ctx, "shutting down NATS connection", "err", err)
	}

	if err := a.echo.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}
