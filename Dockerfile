FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -a -installsuffix cgo \
    -o saiwebsocket \
    ./websockets_pro.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata gettext

RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/saiwebsocket .
COPY --from=builder /build/scripts/docker-entrypoint.sh /usr/local/bin/
COPY --from=builder /build/config.template .

RUN chmod +x /usr/local/bin/docker-entrypoint.sh
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8000

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["./saiwebsocket"]
