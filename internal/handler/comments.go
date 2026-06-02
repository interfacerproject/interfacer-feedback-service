package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/interfacerproject/interfacer-feedback-service/internal/database"
)

// CreateComment handles POST /api/v1/projects/:project_ulid/comments (auth required).
func CreateComment(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectULID := c.Param("project_ulid")
		userULID := c.GetString("user_ulid")

		var body struct {
			ParentID    *string `json:"parent_id"`
			Content     string  `json:"content" binding:"required"`
			Attachments *string `json:"attachments"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if body.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Content must not be empty"})
			return
		}

		comment, err := database.InsertComment(db, projectULID, userULID, body.ParentID, body.Content, body.Attachments)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, comment)
	}
}

// GetComments handles GET /api/v1/projects/:project_ulid/comments (public).
func GetComments(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectULID := c.Param("project_ulid")

		limit := 20
		if l := c.Query("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		var parentID *string
		if pid := c.Query("parent_id"); pid != "" {
			parentID = &pid
		}

		var cursor *int64
		if cur := c.Query("cursor"); cur != "" {
			if parsed, err := strconv.ParseInt(cur, 10, 64); err == nil {
				cursor = &parsed
			}
		}

		comments, err := database.GetCommentsByProject(db, projectULID, parentID, limit, cursor)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comments", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"comments": comments})
	}
}

// DeleteComment handles DELETE /api/v1/comments/:comment_id (auth required).
func DeleteComment(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		commentID := c.Param("comment_id")
		userULID := c.GetString("user_ulid")

		if err := database.SoftDeleteComment(db, commentID, userULID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
