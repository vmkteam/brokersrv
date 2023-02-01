package rpcqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	zm "github.com/vmkteam/zenrpc-middleware"
	"github.com/vmkteam/zenrpc/v2"
)

const StreamName = "BROKERSRV"

const (
	maxAckWait    = 5 * time.Minute
	maxAckPending = 1000
)

type Message struct {
	Request json.RawMessage `json:"request"`
	Header  http.Header     `json:"header"`
}

type RPCQueue struct {
	subject string
	js      nats.JetStreamContext
	srv     zenrpc.Server
	pf      Printf
}

type Printf func(format string, v ...interface{})

var statEvents *prometheus.CounterVec

// New initialize new brokersrv rpc queue.
func New(subject string, js nats.JetStreamContext, srv zenrpc.Server, pf Printf) RPCQueue {
	statEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: subject,
		Subsystem: "rpcqueue",
		Name:      "events_total",
		Help:      "RPC queue events distributions.",
	}, []string{"type"})

	prometheus.MustRegister(statEvents)

	return RPCQueue{
		subject: subject,
		js:      js,
		srv:     srv,
		pf:      pf,
	}
}

// Run subscribe to NATs Streaming subject and process events
func (q *RPCQueue) Run() error {
	_, err := q.js.QueueSubscribe(
		fmt.Sprintf("%s.%s", StreamName, q.subject),
		fmt.Sprintf("group-%s", q.subject), q.handleMessage,
		nats.ManualAck(),
		nats.Durable(fmt.Sprintf("dur-%s", q.subject)),
		nats.BindStream(StreamName),
		nats.MaxAckPending(maxAckPending),
		nats.AckWait(maxAckWait),
	)
	if err != nil {
		return err
	}

	return nil
}

// handleMessage send message to rpc server and acknowledge event.
func (q *RPCQueue) handleMessage(message *nats.Msg) {
	var (
		m         Message
		zenrpcReq zenrpc.Request
	)

	err := json.Unmarshal(message.Data, &m)
	if err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to unmarshal message err=%q", err)
		return
	}

	err = json.Unmarshal(m.Request, &zenrpcReq)
	if err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to unmarshal zenrpc request err=%q", err)
		return
	}

	_, err = q.srv.Do(q.newContext(m.Header), m.Request)
	if err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to send request to rpc server err=%q", err)
		return
	}

	if err = message.Ack(); err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to ack message=%q err=%q", string(message.Data), err)
		return
	}

	statEvents.WithLabelValues("success").Inc()
}

// newContext create new context with data from headers.
func (q *RPCQueue) newContext(h http.Header) context.Context {
	ctx := context.Background()
	ctx = zm.NewIPContext(ctx, "127.0.0.1")
	ctx = zm.NewXRequestIDContext(ctx, h.Get(echo.HeaderXRequestID))
	ctx = zm.NewUserAgentContext(ctx, h.Get("User-Agent"))
	ctx = zm.NewVersionContext(ctx, h.Get("Version"))
	ctx = zm.NewPlatformContext(ctx, h.Get("Platform"))

	return ctx
}
