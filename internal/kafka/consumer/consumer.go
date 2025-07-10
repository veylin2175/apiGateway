package consumer

import (
	"apiGateway/internal/config"
	"apiGateway/internal/dto"
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
	"sync"
)

type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer создает новый консюмер кафки
func NewConsumer(cfg config.Kafka, topic string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     cfg.Brokers,
			Topic:       topic,
			GroupID:     cfg.GroupID,
			StartOffset: kafka.LastOffset,
			MaxBytes:    10e6,
		}),
	}
}

// Run запускает консюмер
func (c *Consumer) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Println("stopping kafka consumer...")
			if err := c.reader.Close(); err != nil {
				log.Printf("failed to close kafka reader: %v", err)
			}
			return
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("kafka read error: %v", err)
				continue
			}

			var user dto.VotingReq
			if err := json.Unmarshal(msg.Value, &user); err != nil {
				log.Printf("failed to unmarshal user: %v", err)
				continue
			}

			log.Printf("Consume user: %v", user)
		}
	}
}
