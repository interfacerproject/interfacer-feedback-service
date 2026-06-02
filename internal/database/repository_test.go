package database

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
)

func TestULIDGeneration(t *testing.T) {
	// ULIDs should be unique, sortable, and 26 characters
	id1 := ulid.Make().String()
	id2 := ulid.Make().String()

	if id1 == id2 {
		t.Error("two generated ULIDs should not be equal")
	}

	if len(id1) != 26 {
		t.Errorf("ULID should be 26 characters, got %d: %s", len(id1), id1)
	}

	if len(id2) != 26 {
		t.Errorf("ULID should be 26 characters, got %d: %s", len(id2), id2)
	}
}

func TestUpsertReview_Insert(t *testing.T) {
	db := setupTestDB(t)

	review, err := UpsertReview(db, "proj-1", "user-1", 4, strPtr("Great project"))
	if err != nil {
		t.Fatalf("UpsertReview (insert) failed: %v", err)
	}

	if review.Rating != 4 {
		t.Errorf("expected rating 4, got %d", review.Rating)
	}
	if review.ProjectULID != "proj-1" {
		t.Errorf("expected project proj-1, got %s", review.ProjectULID)
	}
	if review.UserULID != "user-1" {
		t.Errorf("expected user-1, got %s", review.UserULID)
	}
}

func TestUpsertReview_Update(t *testing.T) {
	db := setupTestDB(t)

	// First insert
	review1, err := UpsertReview(db, "proj-1", "user-1", 3, strPtr("Okay"))
	if err != nil {
		t.Fatalf("first UpsertReview failed: %v", err)
	}

	// Update
	review2, err := UpsertReview(db, "proj-1", "user-1", 5, strPtr("Amazing!"))
	if err != nil {
		t.Fatalf("second UpsertReview failed: %v", err)
	}

	// Same ID, updated rating/content
	if review2.ID != review1.ID {
		t.Errorf("expected same ID on update, got %s vs %s", review2.ID, review1.ID)
	}
	if review2.Rating != 5 {
		t.Errorf("expected updated rating 5, got %d", review2.Rating)
	}

	// Should only have one review per project+user
	reviews, err := GetReviewsByProject(db, "proj-1", 10, nil)
	if err != nil {
		t.Fatalf("GetReviewsByProject failed: %v", err)
	}
	if len(reviews) != 1 {
		t.Errorf("expected 1 review after update, got %d", len(reviews))
	}
}

func TestGetReviewsByProject_Pagination(t *testing.T) {
	db := setupTestDB(t)

	// Insert 3 reviews with staggered timestamps to test cursor pagination
	for i := 1; i <= 3; i++ {
		_, err := UpsertReview(db, "proj-pag", "user-"+string(rune('0'+i)), i, nil)
		if err != nil {
			t.Fatalf("insert review %d failed: %v", i, err)
		}
		// Small delay to ensure distinct created_at timestamps
		time.Sleep(1100 * time.Millisecond)
	}

	// Get first page (limit 2)
	reviews, err := GetReviewsByProject(db, "proj-pag", 2, nil)
	if err != nil {
		t.Fatalf("GetReviewsByProject failed: %v", err)
	}
	if len(reviews) != 2 {
		t.Errorf("expected 2 reviews in first page, got %d", len(reviews))
	}

	// Get second page using cursor
	cursor := reviews[1].CreatedAt.Unix()
	reviews2, err := GetReviewsByProject(db, "proj-pag", 2, &cursor)
	if err != nil {
		t.Fatalf("GetReviewsByProject (cursor) failed: %v", err)
	}
	if len(reviews2) != 1 {
		t.Errorf("expected 1 review in second page, got %d", len(reviews2))
	}
}

func TestGetReviewSummary(t *testing.T) {
	db := setupTestDB(t)

	// Insert ratings: 5, 4, 4 → avg 4.33, count 3
	_, _ = UpsertReview(db, "proj-sum", "u1", 5, nil)
	_, _ = UpsertReview(db, "proj-sum", "u2", 4, nil)
	_, _ = UpsertReview(db, "proj-sum", "u3", 4, nil)

	summary, err := GetReviewSummary(db, "proj-sum")
	if err != nil {
		t.Fatalf("GetReviewSummary failed: %v", err)
	}

	if summary.TotalReviews != 3 {
		t.Errorf("expected 3 total reviews, got %d", summary.TotalReviews)
	}

	avg := summary.AverageRating
	if avg < 4.3 || avg > 4.4 {
		t.Errorf("expected average ≈ 4.33, got %f", avg)
	}

	if summary.Distribution[5] != 1 {
		t.Errorf("expected 1 rating of 5, got %d", summary.Distribution[5])
	}
	if summary.Distribution[4] != 2 {
		t.Errorf("expected 2 ratings of 4, got %d", summary.Distribution[4])
	}
}

func TestSoftDeleteComment(t *testing.T) {
	db := setupTestDB(t)

	comment, err := InsertComment(db, "proj-1", "user-1", nil, "Hello world", nil)
	if err != nil {
		t.Fatalf("InsertComment failed: %v", err)
	}

	// Delete by author
	err = SoftDeleteComment(db, comment.ID, "user-1")
	if err != nil {
		t.Fatalf("SoftDeleteComment failed: %v", err)
	}

	// Deleted comments should not appear in active results
	comments, err := GetCommentsByProject(db, "proj-1", nil, 10, nil)
	if err != nil {
		t.Fatalf("GetCommentsByProject failed: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 active comments after soft delete, got %d", len(comments))
	}

	// Wrong user cannot delete
	comment2, _ := InsertComment(db, "proj-1", "user-2", nil, "Another", nil)
	err = SoftDeleteComment(db, comment2.ID, "user-1")
	if err == nil {
		t.Error("expected error when wrong user tries to delete")
	}
}
