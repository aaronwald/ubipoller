# UbiPoller

A Go application that queries the Ubiquiti Site Manager API for ISP metrics and publishes them to an MQTT broker.

## Features

- Queries Ubiquiti ISP metrics API every 5 minutes (configurable)
- Publishes metrics data to MQTT broker
- Supports command-line configuration
- Structured logging with configurable levels
- Graceful shutdown handling
- Automatic retry and error handling

## Prerequisites

1. **Ubiquiti API Key**: Get your API key from [unifi.ui.com](https://unifi.ui.com/api)
   - Sign in to the UniFi Site Manager
   - Navigate to the API section from the left navigation bar
   - Select "Create API Key"
   - Copy and securely store the generated key

2. **MQTT Broker**: You'll need access to an MQTT broker (e.g., Mosquitto, AWS IoT, etc.)

## Installation

### Build from source

```bash
# Clone the repository
git clone <repository-url>
cd ubipoller

# Build the application
go build -o ubipoller .
```

## Usage

### Basic Usage

```bash
./ubipoller \
  --api-key "your-ubiquiti-api-key" \
  --mqtt-broker "tcp://localhost:1883"
```

### Full Configuration Example

```bash
./ubipoller \
  --api-key "your-ubiquiti-api-key" \
  --api-url "https://api.ui.com/ea/isp-metrics" \
  --metric-type "5m" \
  --mqtt-broker "tcp://mqtt.example.com:1883" \
  --mqtt-client-id "ubipoller-001" \
  --mqtt-topic "home/ubiquiti/isp-metrics" \
  --mqtt-username "mqtt-user" \
  --mqtt-password "mqtt-password" \
  --interval "5m" \
  --log-level "info"
```

### Command Line Options

| Option | Required | Default | Description |
|--------|----------|---------|-------------|
| `--api-key` | Yes | - | Ubiquiti API key for authentication |
| `--api-url` | No | `https://api.ui.com/ea/isp-metrics` | Base URL for Ubiquiti API |
| `--metric-type` | No | `5m` | Metric type to query (5m, 1h, 1d) |
| `--mqtt-broker` | Yes | - | MQTT broker URL (e.g., tcp://localhost:1883) |
| `--mqtt-client-id` | No | `ubipoller` | MQTT client ID |
| `--mqtt-topic` | No | `ubiquiti/isp-metrics` | MQTT topic to publish metrics |
| `--mqtt-username` | No | - | MQTT username (optional) |
| `--mqtt-password` | No | - | MQTT password (optional) |
| `--interval` | No | `5m` | Query interval for fetching metrics |
| `--log-level` | No | `info` | Log level (debug, info, warn, error) |

## Data Format

The application publishes ISP metrics in JSON format to the configured MQTT topic. The data structure includes:

```json
{
  "data": [
    {
      "metricType": "5m",
      "periods": [
        {
          "data": {
            "wan": {
              "avgLatency": 9,
              "download_kbps": 157000,
              "downtime": 0,
              "ispAsn": "33176",
              "ispName": "DTC Cable",
              "maxLatency": 9,
              "packetLoss": 0,
              "upload_kbps": 149000,
              "uptime": 100
            }
          },
          "metricTime": "2025-09-20T17:00:00Z",
          "version": "9.4.19"
        }
      ]
    }
  ]
}
```

## Monitoring and Logging

The application provides structured logging with the following levels:
- `debug`: Detailed debugging information
- `info`: General information about application operation
- `warn`: Warning messages for potential issues
- `error`: Error messages for failures

Example log output:
```
INFO[2025-09-21T10:00:00Z] Starting ubipoller application
INFO[2025-09-21T10:00:00Z] Configuration loaded interval=5m0s metric_type=5m mqtt_topic=ubiquiti/isp-metrics
INFO[2025-09-21T10:00:00Z] Connected to MQTT broker
INFO[2025-09-21T10:00:05Z] Metrics fetched and published successfully
```

## Environment Variables

While the application uses command-line flags, you can also set environment variables if you modify the Kong configuration:

```bash
export UBIQUITI_API_KEY="your-api-key"
export MQTT_BROKER="tcp://localhost:1883"
```

## Docker Usage

You can run the application in a Docker container:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ubipoller .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/ubipoller .
CMD ["./ubipoller"]
```

## Troubleshooting

### Common Issues

1. **API Authentication Errors**: Ensure your API key is valid and hasn't expired
2. **MQTT Connection Issues**: Verify the broker URL, credentials, and network connectivity
3. **Rate Limiting**: The Ubiquiti API has rate limits (100 requests/minute for EA version)

### Debug Mode

Run with debug logging to get detailed information:

```bash
./ubipoller --log-level debug --api-key "your-key" --mqtt-broker "tcp://localhost:1883"
```

## License

This project is licensed under the MIT License.