package authz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDecodePDPResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authz/decide" {
			t.Errorf("path: got %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Response{Decision: "allow", ReasonCode: "ok", Subject: "01PERSONALICE"})
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	resp, err := c.Decide(context.Background(), BuildRequest("r", "feedback.review.create", "pk", Proof{}))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if resp.Decision != "allow" || resp.Subject != "01PERSONALICE" {
		t.Fatalf("decoded: %+v", resp)
	}
}

func TestDecideErrorsOnEmptyBaseURL(t *testing.T) {
	c := &Client{BaseURL: "", HTTP: http.DefaultClient}
	if _, err := c.Decide(context.Background(), Request{}); err == nil {
		t.Fatal("empty base URL must error")
	}
}

func TestDecideErrorsOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := c.Decide(context.Background(), Request{}); err == nil {
		t.Fatal("non-2xx must error")
	}
}
