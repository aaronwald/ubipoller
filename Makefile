.PHONY: build run clean test docker docker-run help

# Default target
help:
	@echo "Available targets:"
	@echo "  build      - Build the application"
	@echo "  run        - Run the application (requires API_KEY and MQTT_BROKER env vars)"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  docker     - Build Docker image"
	@echo "  docker-run - Run with docker-compose"
	@echo "  help       - Show this help message"

# Build the application
build:
	go build -o ubipoller .

# Run the application (requires environment variables)
run: build
	./ubipoller \
		--api-key "$(API_KEY)" \
		--mqtt-broker "$(MQTT_BROKER)" \
		--log-level "$(LOG_LEVEL)"

# Clean build artifacts
clean:
	rm -f ubipoller
	go clean

# Run tests
test:
	go test -v ./...

# Build Docker image
docker:
	docker build -t ubipoller:latest .

# Run with docker-compose
docker-run:
	docker-compose up -d

# Stop docker-compose
docker-stop:
	docker-compose down

# View logs
logs:
	docker-compose logs -f ubipoller