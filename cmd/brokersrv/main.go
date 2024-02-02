package main

import (
	"io"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/vmkteam/brokersrv/pkg/app"

	"github.com/BurntSushi/toml"
	"github.com/namsral/flag"
	"github.com/nats-io/nats.go"
)

const appName = "brokersrv"

var (
	fs           = flag.NewFlagSetWithEnvPrefix(os.Args[0], "BROKERSRV", 0)
	flConfigPath = fs.String("config", "config.toml", "Path to config file")
	flVerbose    = fs.Bool("verbose", false, "enable debug output")
	cfg          app.Config
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	flag.DefaultConfigFlagname = "config.flag"
	exitOnError(fs.Parse(os.Args[1:]))
	fixStdLog(*flVerbose)

	version := appVersion()
	log.Printf("starting %v version=%v", appName, version)
	if _, err := toml.DecodeFile(*flConfigPath, &cfg); err != nil {
		exitOnError(err)
	}

	// connect to NATS cluster
	nc, err := nats.Connect(cfg.NATS.URL, nats.Name(appName), nats.MaxReconnects(100), nats.ReconnectWait(3*time.Second))
	exitOnError(err)

	// create & run app
	application := app.New(appName, cfg, nc)

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

// appVersion returns app version from VCS info
func appVersion() string {
	result := "devel"
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return result
	}

	for _, v := range info.Settings {
		if v.Key == "vcs.revision" {
			result = v.Value
		}
	}

	if len(result) > 8 {
		result = result[:8]
	}

	return result
}
