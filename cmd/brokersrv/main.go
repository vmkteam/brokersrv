package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vmkteam/brokersrv/pkg/app"

	"github.com/BurntSushi/toml"
	"github.com/getsentry/sentry-go"
	"github.com/namsral/flag"
	"github.com/nats-io/nats.go"
	"github.com/vmkteam/appkit"
	"github.com/vmkteam/embedlog"
)

const appName = "brokersrv"

var (
	fs           = flag.NewFlagSetWithEnvPrefix(os.Args[0], "BROKERSRV", 0)
	flConfigPath = fs.String("config", "config.toml", "Path to config file")
	flVerbose    = fs.Bool("verbose", false, "enable debug output")
	flJSONLogs   = fs.Bool("json", false, "enable json output")
	flDev        = fs.Bool("dev", false, "enable dev mode")
	cfg          app.Config
)

func main() {
	flag.DefaultConfigFlagname = "config.flag"
	exitOnError(fs.Parse(os.Args[1:]))

	// setup logger
	sl, ctx := embedlog.NewLogger(*flVerbose, *flJSONLogs), context.Background()
	if *flDev {
		sl = embedlog.NewDevLogger()
	}
	slog.SetDefault(sl.Log()) // set default logger

	version := appkit.Version()
	sl.Print(ctx, "starting", "app", appName, "version", version)
	if _, err := toml.DecodeFile(*flConfigPath, &cfg); err != nil {
		exitOnError(err)
	}

	// enable sentry
	if cfg.Sentry.DSN != "" {
		exitOnError(sentry.Init(sentry.ClientOptions{
			Dsn:         cfg.Sentry.DSN,
			Environment: cfg.Sentry.Environment,
			Release:     version,
		}))
	}

	// connect to NATS cluster
	nc, err := nats.Connect(cfg.NATS.URL, nats.Name(appName), nats.MaxReconnects(100), nats.ReconnectWait(3*time.Second))
	exitOnError(err)

	// create & run app
	a := app.New(appName, sl, cfg, nc)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// run app and send panic to sentry
	go func() {
		defer func() {
			if err := recover(); err != nil {
				sentry.CurrentHub().Recover(err)
				sentry.Flush(time.Second * 3)
				panic(err)
			}
		}()

		er := a.Run(ctx)
		if errors.Is(er, http.ErrServerClosed) {
			er = nil
		}

		// exit after run failed
		a.PrintOrErr(ctx, "server stopped", er)
		quit <- syscall.SIGTERM
	}()

	<-quit

	if err = a.Shutdown(5 * time.Second); err != nil {
		a.Error(ctx, "shutting down service", "err", err)
	}
}

// exitOnError calls log.Fatal if err wasn't nil.
func exitOnError(err error) {
	if err != nil {
		//nolint:sloglint
		slog.Error(err.Error())
		os.Exit(1)
	}
}
