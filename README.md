# brokersrv: JSON-RPC 2.0 to NATS JetStream Gateway

[![Build Status](https://github.com/vmkteam/brokersrv/actions/workflows/go.yml/badge.svg?branch=master)](https://github.com/vmkteam/brokersrv/actions/workflows/go.yml) [![Linter Status](https://github.com/vmkteam/brokersrv/actions/workflows/golangci-lint.yml/badge.svg?branch=master)](https://github.com/vmkteam/brokersrv/actions/workflows/golangci-lint.yml) [![brokersrv rpc queue go Reference](https://pkg.go.dev/badge/github.com/vmkteam/brokersrv.svg)](https://pkg.go.dev/github.com/vmkteam/brokersrv/pkg/rpcqueue)

`brokersrv` is a transparent gateway on top of JSON-RPC 2.0 server that's passes RPC requests to NATS JetStream server.
It uses [zenrpc](https://github.com/vmkteam/zenrpc) package for processing RPC requests.

# How to Use

1. Configure brokersrv via TOML configuration and run it.
2. Send RPC request to brokersrv.
3. Use `github.com/vmkteam/brokersrv/pkg/rpcqueue` package in your RPC server for pulling RPC requests from NATS JetStream server.

# Example
### We have
- `testsrv` test rpc server with zenrpc package as RPC server listen on `localhost:8080/rpc/`.
- `NATS JetStream server` listen on `localhost:4222`.
- `brokersrv` with following configuration:
```toml
[Server]
Host    = "localhost"
Port    = 8071

[NATS]
URL = "nats://localhost:4222"

[Settings]
RpcServices = [ "testsrv" ]
```

### Use brokersrv package in testrpc for processing RPC requests from NATS JetStream Server

```go
...

import (
    "github.com/vmkteam/brokersrv/pkg/rpcqueue"
)

...

nc, err := rpcqueue.NewClient("nats://localhost:4222", "testsrv")

...

rpcQueue := rpcqueue.New("testsrv", nc.JetStreamConn, zenrpcSrv, someLoggerPrintF)
go rpcQueue.Run()

```

### Send test RPC request
Just send RPC request to `localhost:8071/rpc/testsrv/` via Postman/curl. This request will pass into NATS JetStream server.
After that `rpcQueue` in `testsrv` will fetch this request from NATS JetStream server and pass it to own RPC server.
