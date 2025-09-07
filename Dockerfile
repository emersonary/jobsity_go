# ── Builder ─────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS build
WORKDIR /app

# Faster, reproducible builds
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# If your service does outbound HTTPS, you'll need CA certs at runtime
RUN apk add --no-cache ca-certificates

# Cache modules/build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Build a tiny, static binary:
# -trimpath                 -> drop file system paths from binary
# -buildvcs=false           -> omit VCS info
# -tags "netgo,timetzdata"  -> pure-Go DNS; embed tzdata (no OS tz files needed)
# -ldflags "-s -w"          -> strip debug symbols
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -v -trimpath -buildvcs=false -tags "netgo,timetzdata" \
      -ldflags "-s -w" \
      -o /out/flights ./cmd/server

# ── Runtime ─────────────────────────────────────────────────────────
# Smallest: scratch. If you prefer busybox/alpine, switch base below.
FROM scratch

# Copy CA bundle for HTTPS (copied from alpine builder)
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# (Optional) if your app reads zoneinfo from OS instead of timetzdata, copy tzdata
# COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=build /out/flights /usr/local/bin/flights

# Minimal environment
ENV SEARCH_TIMEOUT=4s \
    CACHE_TTL=30s \
    AUTH_USER=demo \
    AUTH_PASS=demo123 \
    JWT_SECRET=change-me

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/flights"]