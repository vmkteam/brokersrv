package rpcqueue

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Config struct {
	URL           string
	MaxReconnects int
	ReconnectWait int // in seconds
	MaxAckWait    int // in seconds
	MaxAckPending int
}

func (c *Config) SetDefaults() {
	if c.MaxReconnects == 0 {
		c.MaxReconnects = 100
	}
	if c.ReconnectWait == 0 {
		c.ReconnectWait = 3
	}
	if c.MaxAckWait == 0 {
		c.MaxAckWait = 5 * 60
	}
	if c.MaxAckPending == 0 {
		c.MaxAckPending = 1000
	}
}

type Client struct {
	nc       *nats.Conn
	legacyJS nats.JetStreamContext
	stream   jetstream.Stream
	config   Config
}

func NewClient(ctx context.Context, cfg Config, appName string) (*Client, error) {
	if cfg.URL == "" {
		return nil, nil
	}
	cfg.SetDefaults()
	nc, err := nats.Connect(
		cfg.URL, nats.Name(appName),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(time.Duration(cfg.ReconnectWait)*time.Second),
	)
	if err != nil {
		return nil, err
	}

	legacyJS, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	stream, err := js.Stream(ctx, StreamName)
	if err != nil {
		return nil, err
	}

	return &Client{
		nc:       nc,
		legacyJS: legacyJS,
		stream:   stream,
		config:   cfg,
	}, nil
}

func (c *Client) Shutdown() error {
	return c.nc.Drain()
}
