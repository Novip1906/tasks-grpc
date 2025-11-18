package kafka

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Novip1906/tasks-grpc/notifications/internal/config"
	"github.com/Novip1906/tasks-grpc/notifications/internal/email"
	"github.com/Novip1906/tasks-grpc/notifications/pkg/logging"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	readers      map[string]*kafka.Reader
	emailService *email.EmailSenderService
	config       config.Kafka
	wg           sync.WaitGroup
	log          *slog.Logger
}

type MessageHandler interface {
	HandleMessage(ctx context.Context, message []byte) error
}

func NewConsumer(config config.Kafka, emailService *email.EmailSenderService, log *slog.Logger) *Consumer {
	return &Consumer{
		readers:      make(map[string]*kafka.Reader),
		emailService: emailService,
		config:       config,
		log:          log,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("Starting Kafka consumers")

	handlers := map[string]MessageHandler{
		c.config.EmailVerificationTopic: &emailVerificationHandler{emailService: c.emailService, log: c.log},
	}

	for topic, handler := range handlers {
		c.log.Debug("Creating reader for topic", "topic", topic)
		reader := c.createReader(topic)
		c.readers[topic] = reader

		c.wg.Add(1)
		go c.consumeTopic(ctx, topic, reader, handler)
	}

	c.log.Info("Kafka consumers started successfully")
	return nil
}

func (c *Consumer) createReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     c.config.Brokers,
		GroupID:     c.config.GroupId,
		Topic:       topic,
		MaxAttempts: 3,
		MaxWait:     10 * time.Second,
		Logger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			c.log.Debug("[KAFKA] "+msg, args...)
		}),
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...interface{}) {
			c.log.Error("[KAFKA-ERROR] "+msg, args...)
		}),
	})
}

func (c *Consumer) consumeTopic(ctx context.Context, topic string, reader *kafka.Reader, handler MessageHandler) {
	defer c.wg.Done()

	c.log.Info("Starting consumer for topic", "topic", topic)
	for {
		select {
		case <-ctx.Done():
			c.log.Info("Stopping consumer for topic", "topic", topic)
			return
		default:
			c.log.Debug("waiting for msg")
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}

				c.log.Error("Error reading message from Kafka",
					"topic", topic,
					logging.Err(err))
				continue
			}

			c.handleMessageWithRetry(ctx, topic, msg, handler)
		}
	}
}

func (c *Consumer) handleMessageWithRetry(ctx context.Context, topic string, msg kafka.Message, handler MessageHandler) {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := handler.HandleMessage(ctx, msg.Value); err != nil {
			lastErr = err
			c.log.Warn("Failed to process message, retrying",
				"topic", topic,
				"attempt", attempt,
				"maxRetries", maxRetries,
				logging.Err(err))

			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}

		c.log.Info("Successfully processed message",
			"topic", topic,
			"partition", msg.Partition,
			"offset", msg.Offset)
		return
	}

	c.log.Error("Failed to process message after all retries",
		"topic", topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		logging.Err(lastErr))
}

func (c *Consumer) Stop() error {
	c.log.Info("Stopping Kafka consumers...")

	var lastErr error
	for topic, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.log.Error("Error closing Kafka reader",
				"topic", topic,
				"error", err)
			lastErr = err
		}
	}

	c.wg.Wait()

	c.log.Info("Kafka consumers stopped")
	return lastErr
}
