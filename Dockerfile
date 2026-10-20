FROM golang:1.27 AS builder

ENV GOPROXY=https://goproxy.cn,direct \
    GO111MODULE=on

WORKDIR /go/cache
COPY go.mod go.sum ./
RUN go mod download

WORKDIR /src
COPY . .

RUN go vet ./... && go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o app ./examples/basic

FROM debian:bookworm-slim AS prod

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

WORKDIR /opt
COPY --from=builder /src/app /opt/app
COPY --from=builder /src/config /opt/config

EXPOSE 80

CMD ["/opt/app"]
