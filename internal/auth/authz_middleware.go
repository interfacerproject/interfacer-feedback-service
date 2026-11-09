package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	b64 "encoding/base64"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/interfacerproject/interfacer-feedback-service/internal/authz"
)

// NewRequestID returns an opaque id for correlating a decision in logs.
func NewRequestID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// AuthzMiddleware authorizes a write by asking the PDP to prove the acting
// subject from the request's own signature. It re-reads the raw body (the
// exact bytes the client signed — real JSON on create, "{}" on delete),
// restores it for the handler, and sends did-pk + did-sign + base64(body) as
// the proof. On allow it sets user_ulid to the PDP-returned proven subject,
// ignoring the self-asserted x-user-id header. On deny or any transport error
// it writes 403 and aborts (fail closed). Identity is proven at the PDP; the
// authn middleware's local signature check remains the 401 floor.
func AuthzMiddleware(client *authz.Client, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unreadable body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		proof := authz.Proof{
			Signature:  c.Request.Header.Get("did-sign"),
			PayloadB64: b64.StdEncoding.EncodeToString(body),
		}
		req := authz.BuildRequest(NewRequestID(), action, c.Request.Header.Get("did-pk"), proof)

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		resp, derr := client.Decide(ctx, req)
		out := authz.Interpret(resp, derr)
		if !out.Allow {
			log.Printf("feedback authz denied: reason=%s action=%s route=%s",
				out.ReasonCode, action, c.FullPath())
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":  "not authorized for this operation",
				"reason": out.ReasonCode,
			})
			return
		}
		c.Set("user_ulid", out.Subject)
		c.Next()
	}
}
