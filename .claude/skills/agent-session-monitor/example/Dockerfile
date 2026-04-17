# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY session_monitor.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o session_monitor .

# Runtime stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/session_monitor .

# Copy default config
COPY config.yaml ./config.yaml

# Create directory for log files
RUN mkdir -p /var/log/higress

# Non-root user for security
RUN addgroup -S monitor && adduser -S monitor -G monitor
RUN chown -R monitor:monitor /app /var/log/higress
USER monitor

# Default log path (can be overridden via env or args)
ENV LOG_FILE=/var/log/higress/access.log
ENV CONFIG_FILE=/app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/session_monitor"]
CMD ["--config", "/app/config.yaml", "--log", "/var/log/higress/access.log"]
