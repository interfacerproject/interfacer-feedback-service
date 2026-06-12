package handler

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/interfacerproject/interfacer-feedback-service/internal/database"
	"github.com/interfacerproject/interfacer-feedback-service/internal/events"
)

// CreateReview handles POST /api/v1/projects/:project_ulid/reviews (auth required).
// Publishes a RatingUpdated event after successful upsert.
func CreateReview(db *sql.DB, publisher events.Publisher) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectULID := c.Param("project_ulid")
		userULID := c.GetString("user_ulid")

		var body struct {
			Rating  int     `json:"rating" binding:"required"`
			Content *string `json:"content"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if body.Rating < 1 || body.Rating > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Rating must be between 1 and 5"})
			return
		}

		review, err := database.UpsertReview(db, projectULID, userULID, body.Rating, body.Content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save review", "details": err.Error()})
			return
		}

		// Emit RatingUpdated event asynchronously
		go func() {
			summary, err := database.GetReviewSummary(db, projectULID)
			if err != nil {
				log.Printf("Failed to get review summary for event: %v", err)
				return
			}
			event := events.RatingUpdatedEvent{
				ProjectULID:   projectULID,
				AverageRating: summary.AverageRating,
				TotalReviews:  summary.TotalReviews,
			}
			if err := publisher.PublishRatingUpdated(event); err != nil {
				log.Printf("Failed to publish RatingUpdated event: %v", err)
			}
		}()

		c.JSON(http.StatusCreated, review)
	}
}

// GetReviews handles GET /api/v1/projects/:project_ulid/reviews (public).
func GetReviews(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectULID := c.Param("project_ulid")

		limit := 20
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		var cursor *int64
		if cur := c.Query("cursor"); cur != "" {
			if parsed, err := strconv.ParseInt(cur, 10, 64); err == nil {
				cursor = &parsed
			}
		}

		reviews, err := database.GetReviewsByProject(db, projectULID, limit, cursor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"reviews": reviews})
	}
}

// DeleteReview handles DELETE /api/v1/reviews/:review_id (auth required).
// Only the review author can delete their own review.
func DeleteReview(db *sql.DB, publisher events.Publisher) gin.HandlerFunc {
	return func(c *gin.Context) {
		reviewID := c.Param("review_id")
		userULID := c.GetString("user_ulid")

		if err := database.DeleteReview(db, reviewID, userULID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

// GetUserReview handles GET /api/v1/projects/:project_ulid/reviews/mine (requires x-user-id header).
// Returns the current user's review for the project, or nil if none exists.
func GetUserReview(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectULID := c.Param("project_ulid")
		userULID := c.GetHeader("x-user-id")

		if userULID == "" {
			c.JSON(http.StatusOK, gin.H{"review": nil})
			return
		}

		review, err := database.GetUserReview(db, projectULID, userULID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user review", "details": err.Error()})
			return
		}

		if review == nil {
			c.JSON(http.StatusOK, gin.H{"review": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"review": review})
	}
}

// GetReviewSummary handles GET /api/v1/projects/:project_ulid/reviews/summary (public).
func GetReviewSummary(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectULID := c.Param("project_ulid")

		summary, err := database.GetReviewSummary(db, projectULID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch review summary", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summary)
	}
}
