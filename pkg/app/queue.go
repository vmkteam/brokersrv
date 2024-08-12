package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/vmkteam/brokersrv/pkg/rpcqueue"

	"github.com/vmkteam/zenrpc/v2"
)

type Message struct {
	Request zenrpc.Request `json:"request"`
	Header  http.Header    `json:"header"`
}

type QueueManager struct {
	js jetstream.JetStream
}

// NewQueueManager returns new QueueManager.
func NewQueueManager(js jetstream.JetStream) *QueueManager {
	return &QueueManager{
		js: js,
	}
}

// Publish prepare and publish message to NATs.
func (m *QueueManager) Publish(ctx context.Context, service string, zenrpcRequest zenrpc.Request, headers http.Header) error {
	message := Message{
		Request: zenrpcRequest,
		Header:  headers,
	}

	bb, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = m.js.Publish(ctx, rpcqueue.StreamName+"."+service, bb, jetstream.WithRetryWait(5*time.Minute))
	return err
}
