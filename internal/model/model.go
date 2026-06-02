package model

import "time"

// Review represents a user's 1-5 star rating and optional text review for a project.
type Review struct {
	ID          string    `json:"id"`
	ProjectULID string    `json:"project_ulid"`
	UserULID    string    `json:"user_ulid"`
	Rating      int       `json:"rating"`
	Content     *string   `json:"content,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ReviewSummary holds aggregated review statistics for a project.
type ReviewSummary struct {
	AverageRating float64        `json:"average_rating"`
	TotalReviews  int            `json:"total_reviews"`
	Distribution  map[int]int    `json:"rating_distribution"` // rating value -> count
}

// Comment represents a user comment with optional nesting (replies) and soft-delete.
type Comment struct {
	ID          string  `json:"id"`
	ProjectULID string  `json:"project_ulid"`
	UserULID    string  `json:"user_ulid"`
	ParentID    *string `json:"parent_id,omitempty"`
	Content     string  `json:"content"`
	Attachments *string `json:"attachments,omitempty"` // JSON array of file IDs/URLs
	Status      string  `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
