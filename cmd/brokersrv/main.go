package main

import (
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vmkteam/brokersrv/pkg/app"

	"github.com/BurntSushi/toml"
	"github.com/namsral/flag"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
)

const appName = "brokersrv"

var (
	fs           = flag.NewFlagSetWithEnvPrefix(os.Args[0], "BROKERSRV", 0)
	flConfigPath = fs.String("config", "config.toml", "Path to config file")
	flVerbose    = fs.Bool("verbose", false, "enable debug output")
	cfg          app.Config
	version      string
)

func main() {
	rand.Seed(time.Now().UnixNano())
	flag.DefaultConfigFlagname = "config.flag"
	exitOnError(fs.Parse(os.Args[1:]))
	fixStdLog(*flVerbose)

	log.Printf("starting %v version=%v", appName, version)
	if _, err := toml.DecodeFile(*flConfigPath, &cfg); err != nil {
		exitOnError(err)
	}

	// connect to NATS Streaming cluster
	sc, err := stan.Connect(cfg.NATS.ClusterID,
		cfg.NATS.ClientID,
		stan.NatsURL(cfg.NATS.URL),
		stan.NatsOptions(nats.Name(appName)),
		stan.Pings(15, 10),
		stan.SetConnectionLostHandler(func(c stan.Conn, reason error) {
			log.Fatalf("Connection lost, reason: %v", reason)
		}),
	)
	exitOnError(err)

	// create & run app
	application := app.New(appName, cfg, sc)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Run
	go func() {
		if err := application.Run(); err != nil {
			exitOnError(err)
		}
	}()
	<-quit
	application.Shutdown(5 * time.Second)
}

// fixStdLog sets additional params to std logger (prefix D, filename & line).
func fixStdLog(verbose bool) {
	log.SetPrefix("D")
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if verbose {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(io.Discard)
	}
}

// exitOnError calls log.Fatal if err wasn't nil.
func exitOnError(err error) {
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
