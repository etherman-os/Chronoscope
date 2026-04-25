# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

# Stage 2: Run
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

RUN adduser -D -u 1000 appuser
RUN mkdir -p /app && chown appuser:appuser /app
USER appuser
WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/server .

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8081/healthz || exit 1

CMD ["./server"]
