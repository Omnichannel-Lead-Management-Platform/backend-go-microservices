package redis_messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/omnichannel/lead_management_service/internal/domain"
)

type RedisEventBus struct {
	client *redis.Client
}

func NewRedisEventBus(client *redis.Client) *RedisEventBus {
	return &RedisEventBus{
		client: client,
	}
}

func (r *RedisEventBus) Publish(ctx context.Context, topic string, event domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Add to Redis Stream
	err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: topic,
		Values: map[string]interface{}{
			"event_data": data,
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to publish to redis stream %s: %w", topic, err)
	}

	return nil
}

func (r *RedisEventBus) Subscribe(ctx context.Context, topic string) (<-chan domain.Event, error) {
	ch := make(chan domain.Event)

	// In a real production system, you would use XReadGroup to allow multiple consumer replicas
	// For this Viva presentation, a simple blocking XRead loop is perfectly acceptable and demonstrates the architecture.
	
	go func() {
		lastID := "$" // Read only new messages from now on
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				// Block for 2 seconds waiting for new events
				streams, err := r.client.XRead(ctx, &redis.XReadArgs{
					Streams: []string{topic, lastID},
					Count:   1,
					Block:   2 * time.Second,
				}).Result()

				if err == redis.Nil {
					// No new messages, loop again
					continue
				} else if err != nil {
					// We only log the error but don't crash the worker, it will retry
					// context canceled will also trigger this but it's handled in the select above
					continue
				}

				for _, stream := range streams {
					for _, msg := range stream.Messages {
						lastID = msg.ID // Update lastID so we don't read this again
						
						eventDataStr, ok := msg.Values["event_data"].(string)
						if !ok {
							log.Printf("Invalid event_data format in redis stream")
							continue
						}

						var event domain.Event
						if err := json.Unmarshal([]byte(eventDataStr), &event); err != nil {
							log.Printf("Failed to unmarshal event from redis: %v", err)
							continue
						}

						ch <- event
					}
				}
			}
		}
	}()

	return ch, nil
}
