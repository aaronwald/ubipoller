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
# Set environment variable defaults (can be overridden at runtime)
ENV MQTT_USERNAME=""
ENV MQTT_PASSWORD=""
ENV UBI_API_KEY=""
ENV MQTT_BROKER="tcp://mqtt:1883"
ENV MQTT_CLIENT_ID="ubipoller-001"
ENV MQTT_TOPIC="mostert/ubiquiti/isp-metrics"
ENV API_URL="https://api.ui.com/ea/isp-metrics"
ENV METRIC_TYPE="5m"
ENV INTERVAL="5m"
ENV LOG_LEVEL="info"

# Use shell form to allow environment variable expansion
CMD ./ubipoller \
  --api-key "${UBI_API_KEY}" \
  --api-url "${API_URL}" \
  --metric-type "${METRIC_TYPE}" \
  --mqtt-broker "${MQTT_BROKER}" \
  --mqtt-client-id "${MQTT_CLIENT_ID}" \
  --mqtt-topic "${MQTT_TOPIC}" \
  --mqtt-username "${MQTT_USERNAME}" \
  --mqtt-password "${MQTT_PASSWORD}" \
  --interval "${INTERVAL}" \
  --log-level "${LOG_LEVEL}"
