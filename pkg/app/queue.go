package app

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vmkteam/brokersrv/pkg/rpcqueue"

	"github.com/nats-io/nats.go"
	"github.com/vmkteam/zenrpc/v2"
)

type Message struct {
	Request zenrpc.Request `json:"request"`
	Header  http.Header    `json:"header"`
}

type QueueManager struct {
	js nats.JetStreamContext
}

// NewQueueManager returns new QueueManager.
func NewQueueManager(js nats.JetStreamContext) *QueueManager {
	return &QueueManager{
		js: js,
	}
}

// Publish prepare and publish message to NATs.
func (m *QueueManager) Publish(service string, zenrpcRequest zenrpc.Request, headers http.Header) error {
	message := Message{
		Request: zenrpcRequest,
		Header:  headers,
	}

	bb, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = m.js.Publish(rpcqueue.StreamName+"."+service, bb, nats.AckWait(5*time.Minute))
	return err
}
