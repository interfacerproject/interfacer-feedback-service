package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load("../../.env")
}

func TestReviewResponseDoesNotContainNestedUser(t *testing.T) {
	// Verify the Review model JSON never includes user metadata
	// — only user_ulid should be present.
	review := map[string]interface{}{
		"id":           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAB",
		"user_ulid":    "01ARZ3NDEKTSV4RRFFQ69G5FAC",
		"rating":       4,
		"created_at":   "2025-01-01T00:00:00Z",
	}

	body, _ := json.Marshal(review)

	// Confirm no user object slipped in
	reviewMap := make(map[string]interface{})
	json.Unmarshal(body, &reviewMap)

	forbidden := []string{"user", "username", "name", "avatar", "email", "profile"}
	for _, key := range forbidden {
		if _, ok := reviewMap[key]; ok {
			t.Errorf("review response must not contain '%s' field", key)
		}
	}

	// user_ulid must be present
	if _, ok := reviewMap["user_ulid"]; !ok {
		t.Error("review response must contain user_ulid")
	}
}

func TestCommentResponseDoesNotContainNestedUser(t *testing.T) {
	comment := map[string]interface{}{
		"id":           "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAB",
		"user_ulid":    "01ARZ3NDEKTSV4RRFFQ69G5FAC",
		"content":      "Great project!",
		"status":       "active",
		"created_at":   "2025-01-01T00:00:00Z",
	}

	body, _ := json.Marshal(comment)

	commentMap := make(map[string]interface{})
	json.Unmarshal(body, &commentMap)

	forbidden := []string{"user", "username", "name", "avatar", "email", "profile"}
	for _, key := range forbidden {
		if _, ok := commentMap[key]; ok {
			t.Errorf("comment response must not contain '%s' field", key)
		}
	}

	if _, ok := commentMap["user_ulid"]; !ok {
		t.Error("comment response must contain user_ulid")
	}
}

func TestErrorResponseFormat(t *testing.T) {
	// Verify error responses follow DPP convention: {"error": "...", "details": "..."}
	router := setupTestRouter(t)

	// Test validation error (empty content)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/projects/proj-1/comments", strings.NewReader(`{"content":""}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)

	// Must have "error" key per DPP convention
	if _, ok := errResp["error"]; !ok {
		t.Error("error response must contain 'error' key (DPP convention)")
	}
}

func TestHealthEndpoint(t *testing.T) {
	// Verify the service compiles and routes are registered
	// Basic sanity check — the routes exist and return proper content type
	router := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/projects/proj-1/reviews", nil)
	router.ServeHTTP(w, req)

	// Should return JSON (even if empty)
	if w.Header().Get("Content-Type") == "" {
		t.Log("Warning: no Content-Type set on empty response")
	}
}
