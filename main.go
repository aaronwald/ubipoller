package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

// CLI represents the command-line interface configuration
type CLI struct {
	// Ubiquiti API configuration
	ApiKey     string `kong:"required,help='Ubiquiti API key for authentication'"`
	ApiURL     string `kong:"default='https://api.ui.com/ea/isp-metrics',help='Base URL for Ubiquiti API'"`
	MetricType string `kong:"default='5m',help='Metric type to query (5m, 1h, 1d)'"`

	// MQTT configuration
	MqttBroker   string `kong:"required,help='MQTT broker URL (e.g., tcp://localhost:1883)'"`
	MqttClientID string `kong:"default='ubipoller',help='MQTT client ID'"`
	MqttTopic    string `kong:"default='ubiquiti/isp-metrics',help='MQTT topic to publish metrics'"`
	MqttUsername string `kong:"help='MQTT username (optional)'"`
	MqttPassword string `kong:"help='MQTT password (optional)'"`

	// Application configuration
	Interval time.Duration `kong:"default='5m',help='Query interval for fetching metrics'"`
	LogLevel string        `kong:"default='info',help='Log level (debug, info, warn, error)'"`
}

// ISPMetrics represents the structure of ISP metrics data
type ISPMetrics struct {
	Data []MetricData `json:"data"`
}

type MetricData struct {
	MetricType string   `json:"metricType"`
	Periods    []Period `json:"periods"`
}

type Period struct {
	Data       PeriodData `json:"data"`
	MetricTime string     `json:"metricTime"`
	Version    string     `json:"version"`
}

type PeriodData struct {
	WAN WANData `json:"wan"`
}

type WANData struct {
	AvgLatency   int    `json:"avgLatency"`
	DownloadKbps int    `json:"download_kbps"`
	Downtime     int    `json:"downtime"`
	ISPAsn       string `json:"ispAsn"`
	ISPName      string `json:"ispName"`
	MaxLatency   int    `json:"maxLatency"`
	PacketLoss   int    `json:"packetLoss"`
	UploadKbps   int    `json:"upload_kbps"`
	Uptime       int    `json:"uptime"`
}

// UbiquitiClient handles API interactions with Ubiquiti
type UbiquitiClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// MQTTPublisher handles MQTT publishing
type MQTTPublisher struct {
	client mqtt.Client
	topic  string
	logger *logrus.Logger
}

// App represents the main application
type App struct {
	cli            *CLI
	ubiquitiClient *UbiquitiClient
	mqttPublisher  *MQTTPublisher
	logger         *logrus.Logger
}

func main() {
	var cli CLI
	kong.Parse(&cli)

	// Initialize logger
	logger := logrus.New()
	level, err := logrus.ParseLevel(cli.LogLevel)
	if err != nil {
		logger.WithError(err).Fatal("Invalid log level")
	}
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Create application
	app, err := NewApp(&cli, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create application")
	}

	// Handle graceful shutdown
	appCtx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Run the application
	if err := app.Run(appCtx); err != nil {
		logger.WithError(err).Fatal("Application failed")
	}

	logger.Info("Application shutdown complete")
}

// NewApp creates a new application instance
func NewApp(cli *CLI, logger *logrus.Logger) (*App, error) {
	// Create Ubiquiti client
	ubiquitiClient := &UbiquitiClient{
		apiKey:  cli.ApiKey,
		baseURL: cli.ApiURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}

	// Create MQTT publisher
	mqttPublisher, err := NewMQTTPublisher(cli, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create MQTT publisher: %w", err)
	}

	return &App{
		cli:            cli,
		ubiquitiClient: ubiquitiClient,
		mqttPublisher:  mqttPublisher,
		logger:         logger,
	}, nil
}

// Run starts the main application loop
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("Starting ubipoller application")
	a.logger.WithFields(logrus.Fields{
		"interval":    a.cli.Interval,
		"metric_type": a.cli.MetricType,
		"mqtt_topic":  a.cli.MqttTopic,
	}).Info("Configuration loaded")

	// Create ticker for periodic execution
	ticker := time.NewTicker(a.cli.Interval)
	defer ticker.Stop()

	// Perform initial fetch
	if err := a.fetchAndPublishMetrics(ctx); err != nil {
		a.logger.WithError(err).Error("Initial metrics fetch failed")
	}

	// Main loop
	for {
		select {
		case <-ctx.Done():
			a.logger.Info("Shutting down application")
			if a.mqttPublisher != nil {
				a.mqttPublisher.Disconnect()
			}
			return nil
		case <-ticker.C:
			if err := a.fetchAndPublishMetrics(ctx); err != nil {
				a.logger.WithError(err).Error("Failed to fetch and publish metrics")
			}
		}
	}
}

// fetchAndPublishMetrics fetches metrics from Ubiquiti API and publishes to MQTT
func (a *App) fetchAndPublishMetrics(ctx context.Context) error {
	a.logger.Debug("Fetching ISP metrics from Ubiquiti API")

	metrics, err := a.ubiquitiClient.GetISPMetrics(ctx, a.cli.MetricType)
	if err != nil {
		return fmt.Errorf("failed to fetch ISP metrics: %w", err)
	}

	a.logger.WithField("periods_count", len(metrics.Data)).Debug("Metrics fetched successfully")

	// Publish to MQTT
	if err := a.mqttPublisher.Publish(metrics); err != nil {
		return fmt.Errorf("failed to publish metrics to MQTT: %w", err)
	}

	a.logger.Info("Metrics fetched and published successfully")
	return nil
}

// GetISPMetrics fetches ISP metrics from the Ubiquiti API
func (c *UbiquitiClient) GetISPMetrics(ctx context.Context, metricType string) (*ISPMetrics, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, metricType)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")

	c.logger.WithField("url", url).Debug("Making API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var metrics ISPMetrics
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &metrics, nil
}

// NewMQTTPublisher creates a new MQTT publisher
func NewMQTTPublisher(cli *CLI, logger *logrus.Logger) (*MQTTPublisher, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cli.MqttBroker)
	opts.SetClientID(cli.MqttClientID)

	if cli.MqttUsername != "" {
		opts.SetUsername(cli.MqttUsername)
	}
	if cli.MqttPassword != "" {
		opts.SetPassword(cli.MqttPassword)
	}

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		logger.WithFields(logrus.Fields{
			"topic":   msg.Topic(),
			"payload": string(msg.Payload()),
		}).Debug("Received message")
	})

	opts.SetOnConnectHandler(func(client mqtt.Client) {
		logger.Info("Connected to MQTT broker")
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		logger.WithError(err).Error("Lost connection to MQTT broker")
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return &MQTTPublisher{
		client: client,
		topic:  cli.MqttTopic,
		logger: logger,
	}, nil
}

// Publish publishes metrics to MQTT
func (p *MQTTPublisher) Publish(metrics *ISPMetrics) error {
	payload, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"topic":        p.topic,
		"payload_size": len(payload),
	}).Debug("Publishing metrics to MQTT")

	token := p.client.Publish(p.topic, 0, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish to MQTT: %w", token.Error())
	}

	return nil
}

// Disconnect disconnects from MQTT broker
func (p *MQTTPublisher) Disconnect() {
	p.logger.Info("Disconnecting from MQTT broker")
	p.client.Disconnect(250)
}
