NAME := brokersrv
PKG := `go list -f {{.Dir}} ./...`

MAIN := cmd/${NAME}/main.go

fmt:
	@goimports -local ${NAME} -l -w $(PKG)

test:
	@go test -v ./...

lint:
	@golangci-lint run -c .golangci.yml

build:
	@CGO_ENABLED=0 go build $(GOFLAGS) -o ${NAME} $(MAIN)

run:
	@echo "Compiling"
	@go run -buildvcs=true $(GOFLAGS) $(MAIN) -config=cfg/local.toml -verbose

mod:
	@go mod tidy
