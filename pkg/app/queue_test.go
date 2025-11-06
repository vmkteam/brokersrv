package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/vmkteam/brokersrv/pkg/rpcqueue"

	"github.com/vmkteam/embedlog"
	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"
)

var (
	testRPCNamespace  = "arith"
	testZenrpcRequest = zenrpc.Request{
		Version: "2.0",
		Method:  testRPCNamespace + "." + testdata.RPC.ArithService.Multiply,
		Params:  json.RawMessage(`{"a":1,"b":2}`),
	}
	testLogger = embedlog.NewDevLogger()
)

func TestQueueManager(t *testing.T) {
	ctx := context.Background()
	err := testApp.qm.Publish(ctx, rpcqueue.StreamName, testRPCSrvSubject, testZenrpcRequest, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	testRPC := zenrpc.NewServer(zenrpc.Options{AllowCORS: true, HideErrorDataField: true})
	testRPC.Use(testRPCMiddleware(t))
	testRPC.Register(testRPCNamespace, &testdata.ArithService{})

	client, err := rpcqueue.NewClient(ctx, rpcqueue.Config{URL: testNatsURL}, testAppName)
	if err != nil {
		panic(err)
	}

	testQueue := rpcqueue.New(testRPCSrvSubject, client, testRPC, testLogger.Print)

	err = testQueue.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
}

func testRPCMiddleware(t *testing.T) zenrpc.MiddlewareFunc {
	return func(h zenrpc.InvokeFunc) zenrpc.InvokeFunc {
		return func(ctx context.Context, method string, params json.RawMessage) zenrpc.Response {
			methodWithNS := zenrpc.NamespaceFromContext(ctx) + "." + method
			if methodWithNS != testZenrpcRequest.Method {
				t.Errorf("RPC method got %s expected %s", methodWithNS, testZenrpcRequest.Method)
			}

			if string(params) != string(testZenrpcRequest.Params) {
				t.Errorf("RPC params got %s expected %s", params, testZenrpcRequest.Params)
			}

			t.Log("RPC request from NATS received successfully")
			return h(ctx, method, params)
		}
	}
}
