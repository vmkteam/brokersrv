NAME := brokersrv

MAIN := cmd/${NAME}/main.go

fmt:
	@golangci-lint fmt

lint:
	@golangci-lint version
	@golangci-lint config verify
	@golangci-lint run

test:
	@go test -v ./...

build:
	@CGO_ENABLED=0 go build $(GOFLAGS) -o ${NAME} $(MAIN)

run:
	@echo "Compiling"
	@go run $(GOFLAGS) $(MAIN) -config=cfg/local.toml -dev

mod:
	@go mod tidy

run-nats:
	@docker run -p 4222:4222 nats -js
