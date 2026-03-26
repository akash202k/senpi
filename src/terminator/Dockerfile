FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go build -o terminator main.go && chmod +x terminator

FROM ghcr.io/openclaw/openclaw:main-slim
COPY --from=builder /app/terminator /usr/local/bin/terminator