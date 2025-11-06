package rpcqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmkteam/appkit"
	"github.com/vmkteam/zenrpc/v2"
)

const (
	LegacyStreamName = "BROKERSRV"
	StreamName       = "BROKERSRV-V2"
)

var (
	statEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "app",
		Subsystem: "rpcqueue",
		Name:      "events_total",
		Help:      "RPC queue events distributions.",
	}, []string{"type", "subject"})
	registerMetricsOnce sync.Once
)

type Message struct {
	Request json.RawMessage `json:"request"`
	Header  http.Header     `json:"header"`
}

type RPCQueue struct {
	subject string
	client  *Client
	srv     *zenrpc.Server
	pf      Print
}

type Print func(ctx context.Context, msg string, args ...any)

// New initialize new brokersrv rpc queue.
func New(subject string, client *Client, srv *zenrpc.Server, p Print) RPCQueue {
	registerMetricsOnce.Do(func() {
		prometheus.MustRegister(statEvents)
	})

	return RPCQueue{
		subject: subject,
		client:  client,
		srv:     srv,
		pf:      p,
	}
}

// LegacyRun subscribe to NATs Streaming subject and process events.
//
// Deprecated: should only be used for migration purposes. See README.md for details.
func (q *RPCQueue) LegacyRun(ctx context.Context) error {
	_, err := q.client.legacyJS.QueueSubscribe(
		fmt.Sprintf("%s.%s", LegacyStreamName, q.subject),
		fmt.Sprintf("group-%s", q.subject), q.legacyMessageHandler(ctx),
		nats.ManualAck(),
		nats.Durable(fmt.Sprintf("dur-%s", q.subject)),
		nats.BindStream(LegacyStreamName),
		nats.MaxAckPending(q.client.config.MaxAckPending),
		nats.AckWait(time.Duration(q.client.config.MaxAckWait)*time.Second),
	)
	return err
}

type ConsumerOpts func(jetstream.ConsumerConfig) jetstream.ConsumerConfig

// Run subscribe to NATs Streaming subject and process events.
func (q *RPCQueue) Run(ctx context.Context) error {
	cfg := jetstream.ConsumerConfig{
		Name:          fmt.Sprintf("dur-%s-v2", q.subject),
		FilterSubject: fmt.Sprintf("%s.%s", StreamName, q.subject),
		Durable:       fmt.Sprintf("dur-%s-v2", q.subject),
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: q.client.config.MaxAckPending,
		AckWait:       time.Duration(q.client.config.MaxAckWait) * time.Second,
		MaxDeliver:    3,
		BackOff:       []time.Duration{time.Second, 5 * time.Second, 30 * time.Second},
	}
	c, err := q.client.stream.CreateOrUpdateConsumer(ctx, cfg)

	if err != nil {
		return err
	}

	_, err = c.Consume(q.messageHandler(ctx))
	if err != nil {
		return err
	}

	return nil
}

// messageHandler send message to rpc server and acknowledge event.
func (q *RPCQueue) legacyMessageHandler(ctx context.Context) nats.MsgHandler {
	return func(message *nats.Msg) {
		var (
			m         Message
			zenrpcReq zenrpc.Request
		)

		err := json.Unmarshal(message.Data, &m)
		if err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to unmarshal message", "err", err)
			return
		}

		err = json.Unmarshal(m.Request, &zenrpcReq)
		if err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to unmarshal zenrpc request", "err", err)
			return
		}

		_, err = q.srv.Do(q.newContext(ctx, m.Header), m.Request)
		if err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to send request to rpc server", "err", err)
			return
		}

		if err = message.Ack(); err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to ack", "message", string(message.Data), "err", err)
			return
		}

		statEvents.WithLabelValues("success", q.subject).Inc()
	}
}

// messageHandler send message to rpc server and acknowledge event.
func (q *RPCQueue) messageHandler(ctx context.Context) jetstream.MessageHandler {
	return func(message jetstream.Msg) {
		var (
			m         Message
			zenrpcReq zenrpc.Request
		)

		err := json.Unmarshal(message.Data(), &m)
		if err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to unmarshal message", "err", err)
			return
		}

		err = json.Unmarshal(m.Request, &zenrpcReq)
		if err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to unmarshal zenrpc request", "err", err)
			return
		}

		_, err = q.srv.Do(q.newContext(ctx, m.Header), m.Request)
		if err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to send request to rpc server", "err", err)
			return
		}

		if err = message.Ack(); err != nil {
			statEvents.WithLabelValues("error", q.subject).Inc()
			q.pf(ctx, "failed to ack", "message", string(message.Data()), "err", err)
			return
		}

		statEvents.WithLabelValues("success", q.subject).Inc()
	}
}

// newContext create new context with data from headers.
func (q *RPCQueue) newContext(ctx context.Context, h http.Header) context.Context {
	ctx = appkit.NewIPContext(ctx, "127.0.0.1")
	ctx = appkit.NewXRequestIDContext(ctx, h.Get(echo.HeaderXRequestID))
	ctx = appkit.NewUserAgentContext(ctx, h.Get("User-Agent"))
	ctx = appkit.NewVersionContext(ctx, h.Get("Version"))
	ctx = appkit.NewPlatformContext(ctx, h.Get("Platform"))

	return ctx
}
