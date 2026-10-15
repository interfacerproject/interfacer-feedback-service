package auth

import (
	"bytes"
	b64 "encoding/base64"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware verifies DID-based EdDSA signatures for write endpoints —
// the authentication floor (401). Skips auth for GET and OPTIONS requests
// (public reads, CORS preflight). It does NOT set "user_ulid": the acting
// subject is proven and set by AuthzMiddleware from the PDP, never from the
// self-asserted x-user-id header.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for public read endpoints and CORS preflight
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// Read the request body for signature verification
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Auth middleware: error reading body: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read request body",
			})
			return
		}
		// Restore the body for downstream handlers
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		// Extract auth headers
		signature := c.Request.Header.Get("did-sign")
		publicKey := c.Request.Header.Get("did-pk")

		if signature == "" || publicKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication failed",
				"details": "Missing authentication headers (did-sign, did-pk)",
			})
			return
		}

		// Build Zenroom auth data
		zenroomData := ZenroomData{
			Gql:            b64.StdEncoding.EncodeToString(body),
			EdDSASignature: signature,
			EdDSAPublicKey: publicKey,
		}

		// Verify DID resolution
		if err := zenroomData.VerifyDid(); err != nil {
			log.Printf("Auth middleware: DID verification failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "DID verification failed",
				"details": err.Error(),
			})
			return
		}

		// Verify EdDSA signature via Zenroom
		if err := zenroomData.IsAuth(); err != nil {
			log.Printf("Auth middleware: Zenroom verification failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication failed",
				"details": err.Error(),
			})
			return
		}

		// Identity is proven and set downstream by AuthzMiddleware (from the
		// PDP), not from the self-asserted x-user-id header. This middleware is
		// only the signature authn floor (401).
		log.Printf("Auth middleware: signature verified for pk %s", publicKey)
		c.Next()
	}
}
