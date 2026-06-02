package handler

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/interfacerproject/interfacer-feedback-service/internal/events"
)

// setupTestRouter creates a Gin engine with all routes registered for contract testing.
func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Register routes (without auth middleware for contract tests)
	// Use a nil DB since we're testing response shapes, not DB interaction
	publisher := &events.LogPublisher{}
	router.POST("/api/v1/projects/:project_ulid/reviews", handlerCreateReviewForTest(publisher))
	router.GET("/api/v1/projects/:project_ulid/reviews", handlerGetReviewsForTest())
	router.GET("/api/v1/projects/:project_ulid/reviews/summary", handlerGetReviewSummaryForTest())
	router.POST("/api/v1/projects/:project_ulid/comments", handlerCreateCommentForTest())
	router.GET("/api/v1/projects/:project_ulid/comments", handlerGetCommentsForTest())
	router.DELETE("/api/v1/comments/:comment_id", handlerDeleteCommentForTest())

	return router
}

// Test stubs — return canned responses for contract shape verification.
// These test the JSON response format, not the database logic.

func handlerCreateReviewForTest(publisher events.Publisher) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Just test the error path — validation
		var body struct {
			Rating  int     `json:"rating" binding:"required"`
			Content *string `json:"content"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
	}
}

func handlerGetReviewsForTest() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"reviews": []interface{}{}})
	}
}

func handlerGetReviewSummaryForTest() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"average_rating":      0.0,
			"total_reviews":       0,
			"rating_distribution": map[int]int{1: 0, 2: 0, 3: 0, 4: 0, 5: 0},
		})
	}
}

func handlerCreateCommentForTest() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Content string `json:"content" binding:"required"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if body.Content == "" {
			c.JSON(400, gin.H{"error": "Content must not be empty"})
			return
		}
	}
}

func handlerGetCommentsForTest() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"comments": []interface{}{}})
	}
}

func handlerDeleteCommentForTest() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "deleted"})
	}
}
