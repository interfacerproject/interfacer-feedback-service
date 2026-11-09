package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/interfacerproject/interfacer-feedback-service/internal/authz"
)

func ctxFor(method, body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, "/api/v1/projects/P1/reviews", strings.NewReader(body))
	req.Header.Set("did-pk", "pkAlice")
	req.Header.Set("did-sign", "SIG")
	req.Header.Set("x-user-id", "VICTIM") // a lie the middleware must ignore
	c.Request = req
	return c, w
}

func stubPDP(t *testing.T, resp authz.Response) *authz.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return &authz.Client{BaseURL: srv.URL, HTTP: srv.Client()}
}

func TestAuthzProvenSubjectOverridesLyingHeader(t *testing.T) {
	c, w := ctxFor(http.MethodPost, `{"rating":5}`)
	mw := AuthzMiddleware(stubPDP(t, authz.Response{Decision: "allow", ReasonCode: "ok", Subject: "ALICE"}), "feedback.review.create")
	mw(c)
	if c.IsAborted() || w.Code == http.StatusForbidden {
		t.Fatalf("allow should proceed: aborted=%v code=%d", c.IsAborted(), w.Code)
	}
	if got := c.GetString("user_ulid"); got != "ALICE" {
		t.Fatalf("user_ulid must be the proven subject, not the header: got %q", got)
	}
}

func TestAuthzBodyRestoredForHandler(t *testing.T) {
	c, _ := ctxFor(http.MethodPost, `{"rating":5}`)
	mw := AuthzMiddleware(stubPDP(t, authz.Response{Decision: "allow", Subject: "ALICE"}), "feedback.review.create")
	mw(c)
	b, _ := io.ReadAll(c.Request.Body)
	if string(b) != `{"rating":5}` {
		t.Fatalf("body must be restored for the handler: got %q", string(b))
	}
}

func TestAuthzDenyIs403(t *testing.T) {
	c, w := ctxFor(http.MethodDelete, `{}`)
	mw := AuthzMiddleware(stubPDP(t, authz.Response{Decision: "deny", ReasonCode: "principal_not_resolved"}), "feedback.review.delete")
	mw(c)
	if !c.IsAborted() || w.Code != http.StatusForbidden {
		t.Fatalf("deny must be 403+abort: aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}

func TestAuthzUnreachableFailsClosed(t *testing.T) {
	c, w := ctxFor(http.MethodDelete, `{}`)
	mw := AuthzMiddleware(&authz.Client{BaseURL: "", HTTP: http.DefaultClient}, "feedback.review.delete")
	mw(c)
	if !c.IsAborted() || w.Code != http.StatusForbidden {
		t.Fatalf("unreachable PDP must fail closed: aborted=%v code=%d", c.IsAborted(), w.Code)
	}
}
