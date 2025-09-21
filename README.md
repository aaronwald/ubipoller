# UbiPoller

A Go application that queries the Ubiquiti Site Manager API for ISP metrics and publishes them to an MQTT broker.

## Features

- Queries Ubiquiti ISP metrics API every 5 minutes (configurable)
- **Site-specific publishing**: Includes siteId in MQTT topic for multi-site deployments
- **Latest latency focus**: Publishes only the most recent latency metrics for each site
- Publishes simplified latency data to MQTT broker with individual topics per site
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

The application publishes **latency-focused metrics** in JSON format to site-specific MQTT topics. Each site gets its own topic in the format: `{base-topic}/{siteId}/latency`

For example, if your base topic is `ubiquiti/isp-metrics`, the published topics will be:
- `ubiquiti/isp-metrics/66f8656d74b8b57aff0b58c3/latency`
- `ubiquiti/isp-metrics/6156282ff71bb3051fd3efb7/latency`

The simplified payload structure for each site contains only the most recent latency data:

```json
{
  "siteId": "66f8656d74b8b57aff0b58c3",
  "hostId": "28704E3BD98300000000082AC0EE000000000899909A00000000668BC714:1416131882",
  "timestamp": "2025-09-21T17:00:00Z",
  "avgLatency": 9,
  "maxLatency": 12,
  "ispName": "DTC Cable",
  "ispAsn": "33176",
  "publishedAt": "2025-09-21T17:05:23.123Z"
}
```

### Benefits of this approach:
- **Multi-site support**: Each site publishes to its own topic
- **Reduced data volume**: Only essential latency metrics are published
- **Real-time focus**: Only the most recent measurement per site
- **Easy filtering**: Subscribe to specific sites: `ubiquiti/isp-metrics/+/latency`
- **Timestamped**: Includes both original metric time and publish time

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
DEBU[2025-09-21T10:00:05Z] Extracted latest latency metrics sites_count=2
DEBU[2025-09-21T10:00:05Z] Publishing latency metric to MQTT avgLatency=9 maxLatency=12 siteId=66f8656d74b8b57aff0b58c3 topic=ubiquiti/isp-metrics/66f8656d74b8b57aff0b58c3/latency
INFO[2025-09-21T10:00:05Z] Latency metrics published successfully sites_published=2
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

### Docker Environment Variables

The Docker image supports the following environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `UBI_API_KEY` | - | **Required**: Ubiquiti API key |
| `MQTT_USERNAME` | - | **Required**: MQTT username |
| `MQTT_PASSWORD` | - | **Required**: MQTT password |
| `MQTT_BROKER` | `tcp://mqtt:1883` | MQTT broker URL |
| `MQTT_CLIENT_ID` | `ubipoller-001` | MQTT client ID |
| `MQTT_TOPIC` | `mostert/ubiquiti/isp-metrics` | Base MQTT topic |
| `API_URL` | `https://api.ui.com/ea/isp-metrics` | Ubiquiti API URL |
| `METRIC_TYPE` | `5m` | Metric type (5m, 1h, 1d) |
| `INTERVAL` | `5m` | Query interval |
| `LOG_LEVEL` | `info` | Log level |

### Docker Run Example

```bash
docker run -d \
  --name ubipoller \
  -e UBI_API_KEY="your-api-key" \
  -e MQTT_USERNAME="your-username" \
  -e MQTT_PASSWORD="your-password" \
  -e MQTT_BROKER="tcp://your-mqtt-broker:1883" \
  ubipoller:latest
```

## Kubernetes Deployment

The application is designed for easy Kubernetes deployment using environment variables.

### 1. Create Secrets

Generate base64 encoded secrets:

```bash
# Use the provided script
./generate-secrets.sh

# Or manually:
echo -n "your-api-key" | base64
echo -n "your-mqtt-username" | base64
echo -n "your-mqtt-password" | base64
```

### 2. Apply Kubernetes Manifests

```bash
# Apply the secret
kubectl apply -f ubipoller-secret.yaml

# Deploy the application
kubectl apply -f k8s-deployment.yaml

# Or use ConfigMap approach
kubectl apply -f k8s-configmap.yaml
```

### 3. Monitor Deployment

```bash
# Check pod status
kubectl get pods -l app=ubipoller

# View logs
kubectl logs -f deployment/ubipoller

# Check published metrics
kubectl logs deployment/ubipoller | grep "Latency metrics published"
```

### Kubernetes Configuration Files

- **`k8s-deployment.yaml`**: Basic deployment with secrets
- **`k8s-configmap.yaml`**: Deployment using ConfigMap for non-sensitive config
- **`generate-secrets.sh`**: Helper script to generate base64 encoded secrets

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