# Multi-stage Dockerfile for the GitActionFlow Go backend.
# Build: docker build -t gitactionflow-backend ./backend
# Run:   needs DATABASE_URL pointing at reachable Postgres.

FROM golang:1.25-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
  && adduser -D -H -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/server /app/server

USER appuser

EXPOSE 8080

ENV APP_ENV=production \
    APP_PORT=8080 \
    LOG_LEVEL=info \
    AUTO_MIGRATE=true

ENTRYPOINT ["/app/server"]
