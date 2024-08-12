package rpcqueue

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	maxReconnects = 100
	reconnectWait = 3 * time.Second
)

type Config struct {
	URL string
}

type Client struct {
	JetStreamConn jetstream.JetStream
	NatsConn      *nats.Conn
}

func NewClient(cfg Config, appName string) (*Client, error) {
	if cfg.URL == "" {
		return nil, nil
	}
	nc, err := nats.Connect(cfg.URL, nats.Name(appName), nats.MaxReconnects(maxReconnects), nats.ReconnectWait(reconnectWait))
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, err
	}

	return &Client{JetStreamConn: js, NatsConn: nc}, nil
}
