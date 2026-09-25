# Multi-stage Dockerfile for the GitActionFlow Go backend.
# Build from repo root:
#   docker build -t gitactionflow-backend -f backend/Dockerfile backend
# Or from backend/:
#   docker build -t gitactionflow-backend .
#
# Render: connect this Dockerfile; set env vars; listen on $PORT.

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

# PORT is injected by Render at runtime. APP_PORT remains a local/dev fallback.
ENV APP_ENV=production \
    LOG_LEVEL=info \
    AUTO_MIGRATE=true

ENTRYPOINT ["/app/server"]
