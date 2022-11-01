package app

import (
	"encoding/json"
	"net/http"

	"github.com/nats-io/stan.go"
	"github.com/vmkteam/zenrpc/v2"
)

type Message struct {
	Request zenrpc.Request `json:"request"`
	Header  http.Header    `json:"header"`
}

type QueueManager struct {
	sc stan.Conn
}

// NewQueueManager returns new QueueManager.
func NewQueueManager(sc stan.Conn) *QueueManager {
	return &QueueManager{
		sc: sc,
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

	return m.sc.Publish(service, bb)
}
