package messaging

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"shingoedge/config"

	kafkago "github.com/segmentio/kafka-go"
)

// Client is the Kafka messaging client.
type Client struct {
	mu       sync.RWMutex
	cfg      *config.MessagingConfig
	kafkaW   *kafkago.Writer
	kafkaR   *kafkago.Reader
	stopChan chan struct{}
}

// NewClient creates a messaging client based on config.
func NewClient(cfg *config.MessagingConfig) *Client {
	return &Client{
		cfg:      cfg,
		stopChan: make(chan struct{}),
	}
}

// Connect establishes the Kafka connection.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("no kafka brokers configured")
	}

	c.kafkaW = &kafkago.Writer{
		Addr:         kafkago.TCP(c.cfg.Kafka.Brokers...),
		Balancer:     &kafkago.LeastBytes{},
		RequiredAcks: kafkago.RequireOne,
	}
	return nil
}

// Publish sends a message to the given topic.
func (c *Client) Publish(topic string, payload []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.kafkaW == nil {
		return fmt.Errorf("kafka writer not initialized")
	}
	return c.kafkaW.WriteMessages(context.Background(), kafkago.Message{
		Topic: topic,
		Value: payload,
	})
}

// PublishEnvelope encodes and publishes a protocol envelope to the given topic.
func (c *Client) PublishEnvelope(topic string, env interface{ Encode() ([]byte, error) }) error {
	data, err := env.Encode()
	if err != nil {
		return fmt.Errorf("encode envelope: %w", err)
	}
	return c.Publish(topic, data)
}

// Subscribe registers a handler for messages on the given topic.
// The consumer goroutine automatically reconnects on errors with
// exponential backoff capped at 5 seconds.
func (c *Client) Subscribe(topic string, handler func(payload []byte)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kafkaW == nil {
		return fmt.Errorf("kafka not connected")
	}
	c.kafkaR = kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: c.cfg.Kafka.Brokers,
		Topic:   topic,
		GroupID: c.cfg.Kafka.GroupID,
	})
	go c.readLoop(topic, handler)
	return nil
}

// readLoop reads messages from Kafka, reconnecting on errors with
// exponential backoff (500ms base, capped at 5s, with ±20% jitter).
func (c *Client) readLoop(topic string, handler func(payload []byte)) {
	const (
		baseBackoff = 500 * time.Millisecond
		maxBackoff  = 5 * time.Second
	)
	backoff := baseBackoff

	for {
		c.mu.RLock()
		reader := c.kafkaR
		c.mu.RUnlock()

		if reader == nil {
			return
		}

		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			// Check if we're shutting down
			select {
			case <-c.stopChan:
				return
			default:
			}

			// Add ±20% jitter to avoid thundering herd
			jittered := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))
			log.Printf("kafka read error: %v, reconnecting in %v", err, jittered.Round(time.Millisecond))

			timer := time.NewTimer(jittered)
			select {
			case <-c.stopChan:
				timer.Stop()
				return
			case <-timer.C:
			}

			// Recreate the reader
			c.mu.Lock()
			if c.kafkaR != nil {
				c.kafkaR.Close()
			}
			c.kafkaR = kafkago.NewReader(kafkago.ReaderConfig{
				Brokers: c.cfg.Kafka.Brokers,
				Topic:   topic,
				GroupID: c.cfg.Kafka.GroupID,
			})
			c.mu.Unlock()

			// Increase backoff for next failure
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on successful read
		backoff = baseBackoff
		handler(msg.Value)
	}
}

// IsConnected returns whether the messaging client is connected.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.kafkaW != nil
}

// Close shuts down the messaging connection.
func (c *Client) Close() {
	// Signal readLoop to stop
	select {
	case <-c.stopChan:
	default:
		close(c.stopChan)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kafkaW != nil {
		c.kafkaW.Close()
		c.kafkaW = nil
	}
	if c.kafkaR != nil {
		c.kafkaR.Close()
		c.kafkaR = nil
	}
}
