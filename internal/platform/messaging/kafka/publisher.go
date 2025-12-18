package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/linkflow-ai/linkflow-ai/internal/shared/events"
)

// EventPublisher publishes events to Kafka
type EventPublisher struct {
	producer sarama.AsyncProducer
	config   *Config
	errors   chan error
}

// Config holds Kafka configuration
type Config struct {
	Brokers []string
	Topic   string
}

// NewEventPublisher creates a new Kafka event publisher
func NewEventPublisher(config *Config) (*EventPublisher, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 5
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Version = sarama.V3_3_1_0

	producer, err := sarama.NewAsyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	publisher := &EventPublisher{
		producer: producer,
		config:   config,
		errors:   make(chan error, 100),
	}

	// Handle producer errors
	go publisher.handleErrors()
	
	// Handle successes
	go publisher.handleSuccesses()

	return publisher, nil
}

// Publish publishes an event
func (p *EventPublisher) Publish(ctx context.Context, event *events.Event) error {
	// Set event metadata
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Extract correlation ID from context
	if correlationID := ctx.Value("correlationID"); correlationID != nil {
		event.Metadata.CorrelationID = correlationID.(string)
	}

	// Serialize event
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Determine topic based on event type
	topic := p.getTopicForEvent(string(event.Type))

	// Create Kafka message
	message := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.AggregateID),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("eventType"),
				Value: []byte(event.Type),
			},
			{
				Key:   []byte("correlationId"),
				Value: []byte(event.Metadata.CorrelationID),
			},
			{
				Key:   []byte("aggregateType"),
				Value: []byte(event.AggregateType),
			},
		},
		Timestamp: event.Timestamp,
	}

	// Send message
	select {
	case p.producer.Input() <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case err := <-p.errors:
		return fmt.Errorf("producer error: %w", err)
	}
}

// PublishBatch publishes multiple events
func (p *EventPublisher) PublishBatch(ctx context.Context, events []*events.Event) error {
	for _, event := range events {
		if err := p.Publish(ctx, event); err != nil {
			return fmt.Errorf("failed to publish event %s: %w", event.ID, err)
		}
	}
	return nil
}

// Close closes the publisher
func (p *EventPublisher) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}
	close(p.errors)
	return nil
}

// handleErrors handles producer errors
func (p *EventPublisher) handleErrors() {
	for err := range p.producer.Errors() {
		select {
		case p.errors <- fmt.Errorf("kafka producer error: %w", err.Err):
		default:
			// Log error if channel is full
			fmt.Printf("Producer error (channel full): %v\n", err.Err)
		}
	}
}

// handleSuccesses handles successful messages
func (p *EventPublisher) handleSuccesses() {
	for msg := range p.producer.Successes() {
		// Log successful message delivery
		fmt.Printf("Message delivered to topic %s [partition=%d, offset=%d]\n",
			msg.Topic, msg.Partition, msg.Offset)
	}
}

// getTopicForEvent maps event types to Kafka topics
func (p *EventPublisher) getTopicForEvent(eventType string) string {
	switch eventType {
	case "workflow.created", "workflow.updated", "workflow.deleted", "workflow.activated":
		return "workflow-events"
	case "execution.started", "execution.completed", "execution.failed":
		return "execution-events"
	case "user.registered", "user.logged_in", "user.updated":
		return "user-events"
	case "auth.login", "auth.logout", "auth.token_refreshed":
		return "auth-events"
	default:
		return "default-events"
	}
}
