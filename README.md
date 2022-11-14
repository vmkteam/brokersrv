# brokersrv: JSON-RPC 2.0 to NATS Streaming Gateway

[![Build Status](https://github.com/vmkteam/brokersrv/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/vmkteam/brokersrv/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/vmkteam/brokersrv.svg)](https://pkg.go.dev/github.com/vmkteam/zenrpc)

`brokersrv` is a transparent gateway on top of JSON-RPC 2.0 server that's passes RPC requests to NATS Streaming server.
It uses [zenrpc](https://github.com/vmkteam/zenrpc) package for processing RPC requests.

# How to Use

1. Configure brokersrv via TOML configuration and run it.
2. Send RPC request to brokersrv.
3. Use `github.com/vmkteam/brokersrv/pkg/rpcqueue` package in your RPC server for pulling RPC requests from NATS Streaming server.

# Example
### We have
- `testsrv` test rpc server with zenrpc package as RPC server listen on `localhost:8080/rpc/`.
- `NATS streaming server` listen on `localhost:4222`.
- `brokersrv` with following configuration:
```toml
[Server]
Host    = "localhost"
Port    = 8071

[NATS]
URL = "nats://localhost:4222"
ClusterID = "test-cluster"
ClientID = "brokersrv"

[Settings]
RpcServices = [ "testsrv" ]
```

### Use brokersrv package in testrpc for processing RPC requests from NATS Streaming Server

```go
...

import (
    "github.com/nats-io/nats.go"
    "github.com/nats-io/stan.go"
    "github.com/vmkteam/brokersrv/pkg/rpcqueue"
)

...

sc, err := stan.Connect("test-cluster", "client-id", stan.NatsURL("nats://localhost:4222"), stan.NatsOptions(nats.Name("testsrv")))

...

testHandler := func(req *zenrpc.Request, resp *zenrpc.Response) bool {
	if req.Method == "test" && resp != nil && resp.Error != nil {
		return false
	}
	return true
}

rpcQ := rpcqueue.New("testsrv", sc, zenrpcSrv, someLoggerPrintF)
rpcQ.SetHandler(testHandler)

go rpcQ.Run()
```

### Send test RPC request
Just send RPC request to `localhost:8071/rpc/testsrv/` via Postman/curl. This request will pass into NATS Streaming server.
After that `rpcQueue` in `testsrv` will fetch this request from NATS Streaming server and pass it to own RPC server.
