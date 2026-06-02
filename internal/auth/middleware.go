package auth

import (
	"bytes"
	b64 "encoding/base64"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware verifies DID-based EdDSA signatures for write endpoints.
// Skips auth for GET and OPTIONS requests (public reads, CORS preflight).
// On success, sets "user_ulid" in the Gin context from the x-user-id header.
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
		userULID := c.Request.Header.Get("x-user-id")

		if signature == "" || publicKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication failed",
				"details": "Missing authentication headers (did-sign, did-pk)",
			})
			return
		}

		if userULID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication failed",
				"details": "Missing x-user-id header",
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

		// Set user ULID in context for downstream handlers
		c.Set("user_ulid", userULID)
		log.Printf("Auth middleware: user %s authenticated successfully", userULID)
		c.Next()
	}
}
