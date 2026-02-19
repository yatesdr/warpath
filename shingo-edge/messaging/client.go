package messaging

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"shingoedge/config"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	kafkago "github.com/segmentio/kafka-go"
)

// Client is the unified messaging client (MQTT or Kafka).
type Client struct {
	mu       sync.RWMutex
	cfg      *config.MessagingConfig
	backend  string
	mqttConn mqtt.Client
	kafkaW   *kafkago.Writer
	kafkaR   *kafkago.Reader
}

// NewClient creates a messaging client based on config.
func NewClient(cfg *config.MessagingConfig) *Client {
	return &Client{
		cfg:     cfg,
		backend: cfg.Backend,
	}
}

// Connect establishes the messaging connection.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.backend {
	case "mqtt":
		return c.connectMQTT()
	case "kafka":
		return c.connectKafka()
	default:
		return fmt.Errorf("unknown messaging backend: %s", c.backend)
	}
}

func (c *Client) connectMQTT() error {
	broker := fmt.Sprintf("tcp://%s:%d", c.cfg.MQTT.Broker, c.cfg.MQTT.Port)
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(c.cfg.MQTT.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("mqtt connect: %w", err)
	}
	c.mqttConn = client
	return nil
}

func (c *Client) connectKafka() error {
	c.kafkaW = &kafkago.Writer{
		Addr:         kafkago.TCP(c.cfg.Kafka.Brokers...),
		Balancer:     &kafkago.LeastBytes{},
		RequiredAcks: kafkago.RequireOne,
	}
	return nil
}

// Publish sends a message to the configured topic.
func (c *Client) Publish(topic string, payload []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	switch c.backend {
	case "mqtt":
		if c.mqttConn == nil || !c.mqttConn.IsConnected() {
			return fmt.Errorf("mqtt not connected")
		}
		token := c.mqttConn.Publish(topic, 1, false, payload)
		token.Wait()
		return token.Error()
	case "kafka":
		if c.kafkaW == nil {
			return fmt.Errorf("kafka writer not initialized")
		}
		return c.kafkaW.WriteMessages(context.Background(), kafkago.Message{
			Topic: topic,
			Value: payload,
		})
	default:
		return fmt.Errorf("unknown backend: %s", c.backend)
	}
}

// PublishEnvelope encodes and publishes a protocol envelope to the given topic.
func (c *Client) PublishEnvelope(topic string, env interface{ Encode() ([]byte, error) }) error {
	data, err := env.Encode()
	if err != nil {
		return fmt.Errorf("encode envelope: %w", err)
	}
	return c.Publish(topic, data)
}

// Subscribe registers a handler for messages on the inbound topic.
func (c *Client) Subscribe(topic string, handler func(payload []byte)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.backend {
	case "mqtt":
		if c.mqttConn == nil {
			return fmt.Errorf("mqtt not connected")
		}
		token := c.mqttConn.Subscribe(topic, 1, func(_ mqtt.Client, msg mqtt.Message) {
			handler(msg.Payload())
		})
		token.Wait()
		return token.Error()
	case "kafka":
		c.kafkaR = kafkago.NewReader(kafkago.ReaderConfig{
			Brokers: c.cfg.Kafka.Brokers,
			Topic:   topic,
			GroupID: c.cfg.MQTT.ClientID, // reuse client ID as consumer group
		})
		go func() {
			for {
				msg, err := c.kafkaR.ReadMessage(context.Background())
				if err != nil {
					log.Printf("kafka read: %v", err)
					return
				}
				handler(msg.Value)
			}
		}()
		return nil
	default:
		return fmt.Errorf("unknown backend: %s", c.backend)
	}
}

// IsConnected returns whether the messaging client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch c.backend {
	case "mqtt":
		return c.mqttConn != nil && c.mqttConn.IsConnected()
	case "kafka":
		return c.kafkaW != nil
	default:
		return false
	}
}

// Close shuts down the messaging connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.mqttConn != nil {
		c.mqttConn.Disconnect(1000)
		c.mqttConn = nil
	}
	if c.kafkaW != nil {
		c.kafkaW.Close()
		c.kafkaW = nil
	}
	if c.kafkaR != nil {
		c.kafkaR.Close()
		c.kafkaR = nil
	}
}
