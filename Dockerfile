FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ubipoller .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/ubipoller .

# Expose no ports as this is a client application

# Run the application
# ENTRYPOINT ["./ubipoller"]

ARG MQTT_USERNAME
ARG MQTT_PASSWORD
ARG UBI_API_KEY

CMD ["./ubipoller", \
  "--api-key", "${UBI_API_KEY}", \
  "--api-url", "https://api.ui.com/ea/isp-metrics", \
  "--metric-type", "5m", \
  "--mqtt-broker", "tcp://mqtt:1883", \
  "--mqtt-client-id", "ubipoller-001", \
  "--mqtt-topic", "mostert/ubiquiti/isp-metrics", \
  "--mqtt-username", "${MQTT_USERNAME}", \
  "--mqtt-password", "${MQTT_PASSWORD}", \
  "--interval", "5m", \
  "--log-level", "info"]
