package main

import (
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/interfacerproject/interfacer-feedback-service/internal/auth"
	"github.com/interfacerproject/interfacer-feedback-service/internal/database"
	"github.com/interfacerproject/interfacer-feedback-service/internal/events"
	"github.com/interfacerproject/interfacer-feedback-service/internal/handler"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("No .env file found, proceeding with environment variables")
	}

	// Initialize SQLite database
	db, err := database.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Set up Gin router with CORS matching DPP configuration
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "did-sign", "did-pk", "x-user-id"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Auth middleware for write endpoints
	authMW := auth.AuthMiddleware()

	// Phase 3: Reviews
	// Event publisher (in-process; replace with broker impl when ready)
	publisher := &events.LogPublisher{}

	router.POST("/api/v1/projects/:project_ulid/reviews", authMW, handler.CreateReview(db, publisher))
	router.GET("/api/v1/projects/:project_ulid/reviews", handler.GetReviews(db))
	router.GET("/api/v1/projects/:project_ulid/reviews/summary", handler.GetReviewSummary(db))

	// Phase 4: Comments
	router.POST("/api/v1/projects/:project_ulid/comments", authMW, handler.CreateComment(db))
	router.GET("/api/v1/projects/:project_ulid/comments", handler.GetComments(db))
	router.DELETE("/api/v1/comments/:comment_id", authMW, handler.DeleteComment(db))

	// TODO: Phase 5 - Event publishing

	log.Println("Feedback service starting on :8081")
	router.Run(":8081")
}
