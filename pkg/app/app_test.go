package app

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/vmkteam/embedlog"
)

var (
	testAppName       = "testbrokersrv"
	testSrvSubject    = "testsrv"
	testRPCSrvSubject = "testrpcsrv"
	testNatsSubjects  = []string{testSrvSubject, testRPCSrvSubject}

	testApp *App
)

var testNatsURL = env("NATS_URL", "nats://localhost:4222")

func env(v, def string) string {
	if r := os.Getenv(v); r != "" {
		return r
	}

	return def
}

func TestMain(m *testing.M) {
	var cfg Config
	cfg.Settings.RPCServices = testNatsSubjects
	cfg.NATS.URL = testNatsURL
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 9984

	nc, err := nats.Connect(
		cfg.NATS.URL, nats.Name(testAppName),
	)
	if err != nil {
		panic(err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		panic(err)
	}

	testApp = New(testAppName, embedlog.Logger{}, cfg, nc)
	testApp.registerHandlers()
	if err = testApp.registerJetStream(context.Background()); err != nil {
		panic(err)
	}
	testApp.qm = NewQueueManager(js)

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
