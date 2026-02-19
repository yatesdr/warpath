package messaging

import (
	"log"
	"sync"
	"time"

	"shingoedge/config"
	"shingoedge/store"
)

// OutboxDrainer periodically sends pending outbox messages.
type OutboxDrainer struct {
	db       *store.DB
	client   *Client
	cfg      *config.MessagingConfig
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewOutboxDrainer creates a new outbox drainer.
func NewOutboxDrainer(db *store.DB, client *Client, cfg *config.MessagingConfig) *OutboxDrainer {
	return &OutboxDrainer{
		db:       db,
		client:   client,
		cfg:      cfg,
		stopChan: make(chan struct{}),
	}
}

// Start begins the outbox drain loop.
func (d *OutboxDrainer) Start() {
	d.wg.Add(1)
	go d.drainLoop()
}

// Stop stops the outbox drain loop.
func (d *OutboxDrainer) Stop() {
	select {
	case <-d.stopChan:
	default:
		close(d.stopChan)
	}
	d.wg.Wait()
}

func (d *OutboxDrainer) drainLoop() {
	defer d.wg.Done()

	interval := d.cfg.OutboxDrainInterval
	if interval <= 0 {
		interval = 5 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.drain()
		}
	}
}

func (d *OutboxDrainer) drain() {
	if !d.client.IsConnected() {
		return
	}

	msgs, err := d.db.ListPendingOutbox(50)
	if err != nil {
		log.Printf("list pending outbox: %v", err)
		return
	}

	for _, msg := range msgs {
		topic := d.cfg.OrdersTopic
		if err := d.client.Publish(topic, msg.Payload); err != nil {
			log.Printf("publish outbox msg %d: %v", msg.ID, err)
			d.db.IncrementOutboxRetries(msg.ID)
			continue
		}
		if err := d.db.AckOutbox(msg.ID); err != nil {
			log.Printf("ack outbox msg %d: %v", msg.ID, err)
		}
	}
}
