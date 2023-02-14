package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/vmkteam/brokersrv/pkg/rpcqueue"
	"github.com/vmkteam/zenrpc/v2"
	"github.com/vmkteam/zenrpc/v2/testdata"
)

var (
	testRpcNamespace  = "arith"
	testZenrpcRequest = zenrpc.Request{
		Version: "2.0",
		Method:  testRpcNamespace + "." + testdata.RPC.ArithService.Multiply,
		Params:  json.RawMessage(`{"a":1,"b":2}`),
	}
)

func TestQueueManager(t *testing.T) {
	err := testApp.qm.Publish(testRpcSrvSubject, testZenrpcRequest, http.Header{})
	if err != nil {
		t.Fatal(err)
	}

	testRpc := zenrpc.NewServer(zenrpc.Options{AllowCORS: true, HideErrorDataField: true})
	testRpc.Use(testRpcMiddleware(t))
	testRpc.Register(testRpcNamespace, &testdata.ArithService{})

	testQueue := rpcqueue.New(testRpcSrvSubject, testRpcQueueClient.JetStreamConn, testRpc, t.Logf)

	err = testQueue.Run()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
}

func testRpcMiddleware(t *testing.T) zenrpc.MiddlewareFunc {
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
