package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/interfacerproject/interfacer-feedback-service/internal/model"
	"github.com/oklog/ulid/v2"
)

// InsertComment creates a new comment with optional parent_id for nested replies.
func InsertComment(db *sql.DB, projectULID, userULID string, parentID *string, content string, attachments *string) (*model.Comment, error) {
	now := time.Now().UTC()
	id := ulid.Make().String()

	_, err := db.Exec(`
		INSERT INTO comments (id, project_ulid, user_ulid, parent_id, content, attachments, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)
	`, id, projectULID, userULID, parentID, content, attachments, now.Unix(), now.Unix())
	if err != nil {
		return nil, fmt.Errorf("insert comment: %w", err)
	}

	return &model.Comment{
		ID:          id,
		ProjectULID: projectULID,
		UserULID:    userULID,
		ParentID:    parentID,
		Content:     content,
		Attachments: attachments,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// GetCommentsByProject returns paginated comments for a project.
// If parentID is nil, returns root comments. Otherwise returns replies to the given parent.
// cursor is the created_at Unix timestamp of the last item from the previous page.
func GetCommentsByProject(db *sql.DB, projectULID string, parentID *string, limit int, cursor *int64) ([]model.Comment, error) {
	if limit <= 0 || limit > 100 {
		limit = defaultPageSize
	}

	query := `SELECT id, project_ulid, user_ulid, parent_id, content, attachments, status, created_at, updated_at
		FROM comments
		WHERE project_ulid = ? AND status = 'active'`
	args := []interface{}{projectULID}

	if parentID == nil {
		query += ` AND parent_id IS NULL`
	} else {
		query += ` AND parent_id = ?`
		args = append(args, *parentID)
	}

	if cursor != nil {
		query += ` AND created_at < ?`
		args = append(args, *cursor)
	}

	query += ` ORDER BY created_at ASC LIMIT ?`
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("get comments: %w", err)
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		var createdAt, updatedAt int64
		if err := rows.Scan(&c.ID, &c.ProjectULID, &c.UserULID, &c.ParentID, &c.Content,
			&c.Attachments, &c.Status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		c.CreatedAt = time.Unix(createdAt, 0).UTC()
		c.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

// SoftDeleteComment marks a comment as deleted. Only the author can delete
// (enforced at the handler level by comparing user_ulid).
// Soft deletion preserves the thread hierarchy.
func SoftDeleteComment(db *sql.DB, commentID, userULID string) error {
	result, err := db.Exec(`
		UPDATE comments SET status = 'deleted', updated_at = ?
		WHERE id = ? AND user_ulid = ?
	`, time.Now().UTC().Unix(), commentID, userULID)
	if err != nil {
		return fmt.Errorf("soft delete comment: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("comment not found or not authorized")
	}

	return nil
}
