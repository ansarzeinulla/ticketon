# BiletFlow API (SRS 8: "Docker-based local or demonstration deployment").
#
# Two stages so the shipped image carries the binary and nothing else - no Go
# toolchain, no module cache, no source. The result is a few tens of megabytes
# rather than a gigabyte, which matters when the whole point is that somebody
# can bring the platform up on a laptop.

FROM golang:1.25-alpine AS build

WORKDIR /src

# Dependencies first: they change far less often than the code, so this layer
# stays cached across almost every rebuild.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO off gives a static binary that runs on the scratch-like base below.
# The version stamps are informational; the build works without them.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /out/api ./cmd/api

FROM alpine:3.20

# ca-certificates for outbound TLS (a real email or payment provider later);
# tzdata because events carry IANA zones and the API renders local times.
RUN apk add --no-cache ca-certificates tzdata curl \
    && adduser -D -u 10001 biletflow

WORKDIR /app
COPY --from=build /out/api /app/api

# Uploaded event banners live here. Declared as a volume so they survive a
# container rebuild - losing them would break every public page that shows one.
RUN mkdir -p /app/data/uploads && chown -R biletflow:biletflow /app
VOLUME ["/app/data"]

USER biletflow
EXPOSE 8080

# The health endpoint checks the database too, so an API that is up but cannot
# reach PostgreSQL is correctly reported as unhealthy rather than ready.
HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=5 \
    CMD curl -fsS http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/api"]
