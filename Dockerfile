# syntax=docker/dockerfile:1.7
# Multi-stage build for both sentinel-api and analytics-worker.
# Select the target with --target=api or --target=worker.
# The build stage compiles both binaries; each runtime stage copies only
# its own binary into a distroless image.
#
# Consumed env vars (no baked-in secrets):
#   PORT (default 8080), BLUEPRINT_DB_HOST/PORT/DATABASE/USERNAME/PASSWORD/SCHEMA,
#   REDIS_HOST, REDIS_PORT, CACHED_RULE_TTL, KAFKA_BROKERS (csv host:port)
# See internal/config/config.go for the canonical list.

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/sentinel-api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/analytics-worker ./cmd/analytics-worker

FROM gcr.io/distroless/static-debian12:nonroot AS api
WORKDIR /
COPY --from=build /out/sentinel-api /sentinel-api
COPY --from=build /src/migrations /migrations
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/sentinel-api"]

FROM gcr.io/distroless/static-debian12:nonroot AS worker
WORKDIR /
COPY --from=build /out/analytics-worker /analytics-worker
COPY --from=build /src/migrations /migrations
USER 65532:65532
ENTRYPOINT ["/analytics-worker"]
