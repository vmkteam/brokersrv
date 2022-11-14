package rpcqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nats-io/stan.go"
	"github.com/prometheus/client_golang/prometheus"
	zm "github.com/vmkteam/zenrpc-middleware"
	"github.com/vmkteam/zenrpc/v2"
)

const (
	stanAckWait     = 5 * time.Minute
	stanMaxInflight = 1000
)

type Message struct {
	Request json.RawMessage `json:"request"`
	Header  http.Header     `json:"header"`
}

type RPCQueue struct {
	subject     string
	sc          stan.Conn
	srv         zenrpc.Server
	pf          Printf
	handlerFunc HandlerFunc
}

type HandlerFunc func(req *zenrpc.Request, resp *zenrpc.Response) bool

type Printf func(format string, v ...interface{})

var statEvents *prometheus.CounterVec

// New initialize new brokersrv rpc queue.
func New(subject string, sc stan.Conn, srv zenrpc.Server, pf Printf) RPCQueue {
	statEvents = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: subject,
		Subsystem: "rpcqueue",
		Name:      "events_total",
		Help:      "RPC queue events distributions.",
	}, []string{"type"})

	prometheus.MustRegister(statEvents)

	return RPCQueue{
		subject: subject,
		sc:      sc,
		srv:     srv,
		pf:      pf,
	}
}

// Run subscribe to NATs Streaming subject and process events.
func (q *RPCQueue) Run() error {
	_, err := q.sc.QueueSubscribe(q.subject, fmt.Sprintf("%s-group", q.subject), q.handleMessage,
		stan.DurableName("dur"),
		stan.SetManualAckMode(),
		stan.AckWait(stanAckWait),
		stan.MaxInflight(stanMaxInflight))
	if err != nil {
		return err
	}

	return nil
}

// SetHandler set func to call it before acknowledging event.
func (q *RPCQueue) SetHandler(handlerFunc HandlerFunc) {
	q.handlerFunc = handlerFunc
}

// handleMessage send message to rpc server and acknowledge event.
func (q *RPCQueue) handleMessage(message *stan.Msg) {
	var (
		m          Message
		zenrpcReq  *zenrpc.Request
		zenrpcResp *zenrpc.Response
	)

	err := json.Unmarshal(message.Data, &m)
	if err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to unmarshal message err=%q", err)
		return
	}

	err = json.Unmarshal(m.Request, zenrpcReq)
	if err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to unmarshal zenrpc request err=%q", err)
		return
	}

	resp, err := q.srv.Do(q.newContext(m.Header), m.Request)
	if err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to send request to rpc server err=%q", err)
		return
	}

	if resp != nil {
		err = json.Unmarshal(resp, zenrpcResp)
		if err != nil {
			statEvents.WithLabelValues("error").Inc()
			q.pf("failed to unmarshal zenrpc response err=%q", err)
			return
		}

		if q.handlerFunc != nil && !q.handlerFunc(zenrpcReq, zenrpcResp) {
			statEvents.WithLabelValues("error").Inc() // TODO send error metric if rpc error received?
			return
		}
	}

	if err = message.Ack(); err != nil {
		statEvents.WithLabelValues("error").Inc()
		q.pf("failed to ack message=%q err=%q", message.String(), err)
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
