NAME := brokersrv
PKG := `go list -f {{.Dir}} ./...`

MAIN := cmd/${NAME}/main.go

VERSION?=$(git version > /dev/null 2>&1 && git describe --dirty=-dirty --always 2>/dev/null || echo NO_VERSION)
LDFLAGS=-ldflags "-X=main.version=$(VERSION)"

fmt:
	@goimports -local ${NAME} -l -w $(PKG)

lint:
	@golangci-lint run -c .golangci.yml

build:
	@CGO_ENABLED=0 go build $(LDFLAGS) $(GOFLAGS) -o ${NAME} $(MAIN)

run:
	@echo "Compiling"
	@go run -buildvcs=true $(LDFLAGS) $(GOFLAGS) $(MAIN) -config=cfg/local.toml -verbose

mod:
	@go mod tidy
