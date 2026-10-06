# BiletFlow API image. Two stages: the shipped image carries the binary and
# nothing else - no Go toolchain, no module cache, no source.

FROM golang:1.25-alpine AS build
WORKDIR /src

# Dependencies first: they change less often than the code, so this layer stays
# cached across most rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl && adduser -D -u 10001 biletflow
WORKDIR /app
COPY --from=build /out/api /app/api
USER biletflow
EXPOSE 8080

# /health checks the database too, so an API that cannot reach PostgreSQL is
# reported unhealthy rather than ready.
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=5 \
    CMD curl -fsS http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]
