package events

import "time"

// RatingUpdatedEvent is published when a review is created or updated.
type RatingUpdatedEvent struct {
	EventType    string  `json:"event_type"`
	ProjectULID  string  `json:"project_ulid"`
	AverageRating float64 `json:"average_rating"`
	TotalReviews int     `json:"total_reviews"`
	Timestamp    int64   `json:"timestamp"`
}

// Publisher defines the interface for event publishing.
// Implementations can use RabbitMQ, NATS, Kafka, or an in-process channel.
type Publisher interface {
	PublishRatingUpdated(event RatingUpdatedEvent) error
}

// LogPublisher is a simple in-process publisher that logs events.
// Can be replaced with a real message broker implementation.
type LogPublisher struct{}

func (p *LogPublisher) PublishRatingUpdated(event RatingUpdatedEvent) error {
	event.EventType = "RatingUpdated"
	event.Timestamp = time.Now().UTC().Unix()
	// TODO: Replace with actual message broker publish when ready
	return nil
}
