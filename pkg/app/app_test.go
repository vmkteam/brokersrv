package app

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/vmkteam/brokersrv/pkg/rpcqueue"
)

var (
	testAppName       = "testbrokersrv"
	testSrvSubject    = "testsrv"
	testRpcSrvSubject = "testrpcsrv"
	testNatsSubjects  = []string{testSrvSubject, testRpcSrvSubject}

	testApp            *App
	testRpcQueueClient *rpcqueue.Client
)

var testNatsUrl = env("NATS_URL", "nats://localhost:4222")

func env(v, def string) string {
	if r := os.Getenv(v); r != "" {
		return r
	}

	return def
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UTC().UnixNano())

	var cfg Config
	cfg.Settings.RpcServices = testNatsSubjects
	cfg.NATS.URL = testNatsUrl
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 9984

	nc, err := rpcqueue.NewClient(rpcqueue.Config{URL: cfg.NATS.URL}, testAppName)
	if err != nil {
		panic(err)
	}
	testRpcQueueClient = nc

	testApp = New(testAppName, cfg, testRpcQueueClient.NatsConn)
	testApp.registerHandlers()
	if err = testApp.registerJetStream(); err != nil {
		panic(err)
	}
	testApp.qm = NewQueueManager(testRpcQueueClient.JetStreamConn)

	runTests := m.Run()
	os.Exit(runTests)

}

func TestApp(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testApp.echo.ServeHTTP))
	defer ts.Close()

	var tc = []struct {
		url     string
		in, out string
	}{
		{
			url: fmt.Sprintf("%s/rpc/not-exists/", ts.URL),
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": {"a": 1, "b": 2}, id": 1 }`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"service not exists"}}
`,
		},
		{
			url: fmt.Sprintf("%s/rpc/%s/", ts.URL, testSrvSubject),
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": {"a": 1, "b": 2}, "id": 1 }`,
			out: `{"jsonrpc":"2.0","id":null,"error":{"code":-32602,"message":"request ID not empty"}}
`,
		},
		{
			url: fmt.Sprintf("%s/rpc/%s/", ts.URL, testSrvSubject),
			in:  `{"jsonrpc": "2.0", "method": "arith.multiply", "params": {"a": 1, "b": 2} }`,
			out: `null
`,
		},
	}
	for _, c := range tc {
		res, err := http.Post(c.url, "application/json", bytes.NewBufferString(c.in))
		if err != nil {
			t.Fatal(err)
		}

		resp, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if string(resp) != c.out {
			t.Errorf("Input: %s\n got %s expected %s", c.in, resp, c.out)
		}
	}

}
