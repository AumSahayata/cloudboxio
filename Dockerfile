# syntax=docker/dockerfile:1

# ============================ Build stage ============================
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Download dependencies first so this layer is cached unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Build the binary.
COPY . .
# modernc.org/sqlite is a pure-Go driver, so we build a fully static binary
# with CGO disabled. -trimpath + -ldflags "-s -w" keep the binary small.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cloudboxio .

# =========================== Runtime stage ==========================
# Small image that still ships a shell for easy debugging. For a stricter
# footprint you can swap this for: gcr.io/distroless/static-debian12:nonroot
FROM alpine:3.20

# Run as an unprivileged user and keep all state in a writable data dir.
RUN addgroup -S app && adduser -S app -G app \
 && mkdir -p /data && chown app:app /data \
 && apk add --no-cache ca-certificates

COPY --from=builder /out/cloudboxio /usr/local/bin/cloudboxio

USER app
WORKDIR /data

# On first run the app generates .env, data.db, logs/ and uploads/ in the
# working directory. Mount a volume here so that state survives restarts.
VOLUME ["/data"]

EXPOSE 3000

ENTRYPOINT ["/usr/local/bin/cloudboxio"]
