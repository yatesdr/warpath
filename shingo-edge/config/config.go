package config

import (
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	mu sync.Mutex `yaml:"-"`

	Namespace    string        `yaml:"namespace"`
	LineID       string        `yaml:"line_id"`
	DatabasePath string        `yaml:"database_path"`
	PollRate     time.Duration `yaml:"poll_rate"`

	WarLink   WarLinkConfig   `yaml:"warlink"`
	Web       WebConfig       `yaml:"web"`
	Messaging MessagingConfig `yaml:"messaging"`
	Counter   CounterConfig   `yaml:"counter"`
}

// WarLinkConfig defines the WarLink connection.
type WarLinkConfig struct {
	URL      string        `yaml:"url"        json:"url"`
	PollRate time.Duration `yaml:"poll_rate"   json:"poll_rate"`
	Enabled  bool          `yaml:"enabled"     json:"enabled"`
}

// WebConfig defines the web server settings.
type WebConfig struct {
	Host          string `yaml:"host"`
	Port          int    `yaml:"port"`
	SessionSecret string `yaml:"session_secret"`
	AutoConfirm   bool   `yaml:"auto_confirm"`
}

// MessagingConfig defines the messaging backend.
type MessagingConfig struct {
	Backend            string        `yaml:"backend"` // "mqtt" or "kafka"
	MQTT               MQTTConfig    `yaml:"mqtt"`
	Kafka              KafkaConfig   `yaml:"kafka"`
	OrderTopic         string        `yaml:"order_topic"`
	InboundTopic       string        `yaml:"inbound_topic"`
	DispatchTopic      string        `yaml:"dispatch_topic"`
	OrdersTopic        string        `yaml:"orders_topic"`
	OutboxDrainInterval time.Duration `yaml:"outbox_drain_interval"`
	NodeID             string        `yaml:"node_id"`
}

// MQTTConfig defines MQTT broker settings.
type MQTTConfig struct {
	Broker   string `yaml:"broker"`
	Port     int    `yaml:"port"`
	ClientID string `yaml:"client_id"`
}

// KafkaConfig defines Kafka broker settings.
type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
}

// CounterConfig defines counter anomaly thresholds.
type CounterConfig struct {
	JumpThreshold int64 `yaml:"jump_threshold"`
}

// Defaults returns a Config with sane defaults.
func Defaults() *Config {
	return &Config{
		Namespace:    "plant-a",
		LineID:       "line-1",
		DatabasePath: "shingoedge.db",
		PollRate:     time.Second,
		WarLink: WarLinkConfig{
			URL:      "http://localhost:8080/api",
			PollRate: 2 * time.Second,
			Enabled:  true,
		},
		Web: WebConfig{
			Host: "0.0.0.0",
			Port: 8081,
		},
		Messaging: MessagingConfig{
			Backend:            "mqtt",
			OrderTopic:         "shingoedge/orders",
			InboundTopic:       "shingoedge/dispatch",
			DispatchTopic:      "shingo/dispatch",
			OrdersTopic:        "shingo/orders",
			OutboxDrainInterval: 5 * time.Second,
			MQTT: MQTTConfig{
				Broker: "localhost",
				Port:   1883,
			},
		},
		Counter: CounterConfig{
			JumpThreshold: 1000,
		},
	}
}

// Load reads a YAML config file. If the file doesn't exist, defaults are used.
func Load(path string) (*Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to a YAML file.
func (c *Config) Save(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// NodeID returns the configured node ID, or derives one from namespace.line_id.
func (c *Config) NodeID() string {
	if c.Messaging.NodeID != "" {
		return c.Messaging.NodeID
	}
	return c.Namespace + "." + c.LineID
}

// Lock acquires the config mutex for multi-step mutations.
func (c *Config) Lock() { c.mu.Lock() }

// Unlock releases the config mutex.
func (c *Config) Unlock() { c.mu.Unlock() }
