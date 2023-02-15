FROM golang:1.18-alpine AS builder
COPY . /build
RUN cd /build && go install -mod=mod ./cmd/brokersrv

FROM alpine:latest

ENV TZ=Europe/Moscow
RUN apk --no-cache add ca-certificates tzdata && cp -r -f /usr/share/zoneinfo/$TZ /etc/localtime

COPY --from=builder /go/bin/brokersrv .

ENTRYPOINT ["/brokersrv"]
EXPOSE 8071
