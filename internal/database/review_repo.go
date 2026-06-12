package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/interfacerproject/interfacer-feedback-service/internal/model"
	"github.com/oklog/ulid/v2"
)

const defaultPageSize = 20

// UpsertReview inserts or updates a review. Uses UPDATE-then-INSERT to avoid
// SQLite's INSERT OR REPLACE (which would delete and re-insert, losing the ID).
// Returns the created/updated review with generated ID and timestamps.
func UpsertReview(db *sql.DB, projectULID, userULID string, rating int, content *string) (*model.Review, error) {
	now := time.Now().UTC()

	// Try UPDATE first
	result, err := db.Exec(`
		UPDATE reviews SET rating = ?, content = ?, updated_at = ?
		WHERE project_ulid = ? AND user_ulid = ?
	`, rating, content, now.Unix(), projectULID, userULID)
	if err != nil {
		return nil, fmt.Errorf("upsert review (update): %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	var id string
	var createdAt time.Time

	if rowsAffected > 0 {
		// Updated existing row — fetch ID and original created_at
		var createdUnix int64
		err := db.QueryRow(`
			SELECT id, created_at FROM reviews WHERE project_ulid = ? AND user_ulid = ?
		`, projectULID, userULID).Scan(&id, &createdUnix)
		if err != nil {
			return nil, fmt.Errorf("fetch updated review: %w", err)
		}
		createdAt = time.Unix(createdUnix, 0).UTC()
	} else {
		// Insert new row
		id = ulid.Make().String()
		createdAt = now
		_, err := db.Exec(`
			INSERT INTO reviews (id, project_ulid, user_ulid, rating, content, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, id, projectULID, userULID, rating, content, createdAt.Unix(), now.Unix())
		if err != nil {
			return nil, fmt.Errorf("upsert review (insert): %w", err)
		}
	}

	return &model.Review{
		ID:          id,
		ProjectULID: projectULID,
		UserULID:    userULID,
		Rating:      rating,
		Content:     content,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}, nil
}

// GetReviewsByProject returns paginated reviews for a project, ordered by newest first.
// cursor is the created_at Unix timestamp of the last item from the previous page.
func GetReviewsByProject(db *sql.DB, projectULID string, limit int, cursor *int64) ([]model.Review, error) {
	if limit <= 0 || limit > 100 {
		limit = defaultPageSize
	}

	var rows *sql.Rows
	var err error

	if cursor != nil {
		rows, err = db.Query(`
			SELECT id, project_ulid, user_ulid, rating, content, created_at, updated_at
			FROM reviews
			WHERE project_ulid = ? AND created_at < ?
			ORDER BY created_at DESC
			LIMIT ?
		`, projectULID, *cursor, limit)
	} else {
		rows, err = db.Query(`
			SELECT id, project_ulid, user_ulid, rating, content, created_at, updated_at
			FROM reviews
			WHERE project_ulid = ?
			ORDER BY created_at DESC
			LIMIT ?
		`, projectULID, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("get reviews: %w", err)
	}
	defer rows.Close()

	var reviews []model.Review
	for rows.Next() {
		var r model.Review
		var createdAt, updatedAt int64
		if err := rows.Scan(&r.ID, &r.ProjectULID, &r.UserULID, &r.Rating, &r.Content, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		r.CreatedAt = time.Unix(createdAt, 0).UTC()
		r.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		reviews = append(reviews, r)
	}

	return reviews, rows.Err()
}

// DeleteReview removes a review by ID. Only the author (userULID) can delete.
func DeleteReview(db *sql.DB, reviewID string, userULID string) error {
	result, err := db.Exec(`
		DELETE FROM reviews WHERE id = ? AND user_ulid = ?
	`, reviewID, userULID)
	if err != nil {
		return fmt.Errorf("delete review: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("review not found or not owned by user")
	}

	return nil
}

// GetUserReview returns the review by a specific user for a project, or nil if none exists.
func GetUserReview(db *sql.DB, projectULID string, userULID string) (*model.Review, error) {
	var r model.Review
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, project_ulid, user_ulid, rating, content, created_at, updated_at
		FROM reviews
		WHERE project_ulid = ? AND user_ulid = ?
	`, projectULID, userULID).Scan(&r.ID, &r.ProjectULID, &r.UserULID, &r.Rating, &r.Content, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user review: %w", err)
	}

	r.CreatedAt = time.Unix(createdAt, 0).UTC()
	r.UpdatedAt = time.Unix(updatedAt, 0).UTC()

	return &r, nil
}

// GetReviewSummary returns aggregated stats for a project's reviews.
func GetReviewSummary(db *sql.DB, projectULID string) (*model.ReviewSummary, error) {
	var avgRating sql.NullFloat64
	var totalReviews int

	err := db.QueryRow(`
		SELECT AVG(rating), COUNT(*) FROM reviews WHERE project_ulid = ?
	`, projectULID).Scan(&avgRating, &totalReviews)
	if err != nil {
		return nil, fmt.Errorf("get review summary: %w", err)
	}

	// Rating distribution (1-5)
	distribution := make(map[int]int)
	for rating := 1; rating <= 5; rating++ {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM reviews WHERE project_ulid = ? AND rating = ?
		`, projectULID, rating).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("get rating distribution: %w", err)
		}
		distribution[rating] = count
	}

	avg := 0.0
	if avgRating.Valid {
		avg = avgRating.Float64
	}

	return &model.ReviewSummary{
		AverageRating: avg,
		TotalReviews:  totalReviews,
		Distribution:  distribution,
	}, nil
}
