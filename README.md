# Interfacer Feedback & Interaction Service

![Go Version](https://img.shields.io/badge/Go-1.25.0-blue.svg)
![SQLite](https://img.shields.io/badge/SQLite-3.x-green.svg)
![License](https://img.shields.io/badge/License-AGPL--3.0-yellow.svg)

<br>
<a href="https://www.interfacerproject.eu/">
  <img alt="Interfacer project" src="https://dyne.org/images/projects/Interfacer_logo_color.png" width="350" />
</a>
<br>

A specialized microservice for managing user feedback, project reviews, and threaded comments within the Interfacer Project ecosystem. Built with Go, Gin framework, and SQLite, this service handles project ratings (1–5 stars), aggregated review summaries, hierarchical comment threads with attachments, and event-driven rating updates — all secured with DID-based EdDSA authentication.

> ⚠️ **Early Development Stage**: This microservice is currently in early stage development and is being actively developed as part of the broader Interfacer Project infrastructure.

## 📋 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Architecture](#-architecture)
- [Prerequisites](#-prerequisites)
- [Installation](#-installation)
- [Configuration](#️-configuration)
- [Authentication](#-authentication)
- [API Documentation](#-api-documentation)
- [Data Model](#-data-model)
- [Usage Examples](#-usage-examples)
- [Events](#-events)
- [Development](#️-development)
- [Testing](#-testing)
- [Contributing](#-contributing)
- [License](#-license)

## 🌟 Overview

The Interfacer Feedback & Interaction microservice is a core component of the [Interfacer Project](https://www.interfacerproject.eu/) ecosystem, designed to handle all user-generated feedback and interaction features. This microservice operates within a distributed architecture where:

- **interfacer-gui** (frontend) consumes review scores, comment threads, and summary data
- **interfacer-proxy** acts as the API gateway and routing layer, forwarding signed requests with DID authentication headers
- **interfacer-feedback-service** (this service) manages reviews, ratings, and comments
- **interfacer-dpp** manages Digital Product Passport data independently
- **Project Service** consumes `RatingUpdated` events to cache aggregated review scores

The service supports community-driven quality assessment and conversation by providing:

- **Star Ratings**: 1–5 star reviews tied to projects, with one review per user per project enforced at the database level
- **Review Summaries**: Aggregated average ratings, total counts, and per-rating distributions for display on project cards
- **Threaded Comments**: Hierarchical comment system with nested replies and optional file attachments
- **Event Publishing**: Asynchronous `RatingUpdated` events so the main Project Service can update its cached aggregates in real time

### Ecosystem Position

```
┌───────────────────┐
│  interfacer-gui   │  ← Frontend (browser)
└────────┬──────────┘
         │
    ┌────▼─────┐
    │  Proxy   │  ← DID auth + routing (x-user-id header)
    └────┬─────┘
         │
    ┌────▼──────────────┐
    │ Feedback Service  │  ← Reviews, comments, events
    └────┬──────────────┘
         │ RatingUpdated events
    ┌────▼──────────────┐
    │ Project Service   │  ← Caches aggregated ratings
    └───────────────────┘
```

## ✨ Features

- **Star Ratings & Reviews**: One review per user per project with 1–5 star ratings, optional review text, and on-the-fly updates via upsert
- **Review Summaries**: Aggregated statistics including average rating, total review count, and per-rating distribution histogram
- **Threaded Comments**: Hierarchical comment threads with arbitrary nesting depth, soft deletion that preserves thread structure, and optional file attachment references
- **DID-Based Authentication**: EdDSA signature verification via Zenroom cryptographic VM with DID resolution — identical flow to interfacer-dpp
- **Cursor-Based Pagination**: All list endpoints support limit + cursor pagination for efficient infinite-scroll UIs
- **Event Publishing**: Asynchronous `RatingUpdated` events for inter-service communication (extensible to RabbitMQ, NATS, Kafka)
- **SQLite Storage**: Zero-dependency embedded database with WAL mode for concurrent read performance
- **ULID Identifiers**: Universally unique lexicographically sortable IDs for all primary keys and external references
- **RESTful API**: Clean JSON endpoints matching interfacer-dpp conventions for error responses and CORS configuration
- **Public Reads, Auth Writes**: All `GET` endpoints are public; `POST`, `PUT`, and `DELETE` require DID-authenticated requests

## 🏗️ Architecture

```
interfacer-feedback-service/
├── cmd/
│   └── main/
│       └── main.go           # Application entry point, route registration, CORS setup
├── internal/
│   ├── auth/
│   │   ├── auth.go           # DID verification (Zenroom + DID resolution)
│   │   ├── middleware.go     # Gin auth middleware (skips GET/OPTIONS)
│   │   └── zenflows-crypto/  # Embedded Zenroom verify script (git submodule)
│   ├── database/
│   │   ├── database.go       # SQLite connection (sync.Once singleton), migrations
│   │   ├── review_repo.go    # Review CRUD + summary aggregation queries
│   │   └── comment_repo.go   # Comment CRUD + soft delete queries
│   ├── events/
│   │   └── events.go         # Event publisher interface + RatingUpdatedEvent schema
│   ├── handler/
│   │   ├── reviews.go        # POST/GET/DELETE reviews, GET review summary, GET user's review
│   │   └── comments.go       # POST/GET/DELETE comments
│   └── model/
│       └── model.go          # Data structures: Review, ReviewSummary, Comment
├── data/                     # SQLite database files (gitignored)
├── .env.example              # Environment variable template
├── go.mod                    # Go module dependencies
├── go.sum                    # Go module checksums
└── README.md                 # This file
```

### Technology Stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Language** | Go 1.25.0 | Performance, concurrency, matching interfacer-dpp |
| **Web Framework** | Gin HTTP | Fast, lightweight, matching interfacer-dpp |
| **Database** | SQLite via `modernc.org/sqlite` | Pure Go, zero CGO, embedded, self-contained |
| **Authentication** | Zenroom VM + DID resolution | EdDSA signature verification, identical to interfacer-dpp |
| **Identifiers** | ULID (`github.com/oklog/ulid/v2`) | Sortable, URL-safe, matching interfacer-dpp |
| **CORS** | `github.com/gin-contrib/cors` | Matching interfacer-dpp CORS configuration |
| **Env Loading** | `github.com/joho/godotenv` | Matching interfacer-dpp pattern |

### Design Decisions

- **SQLite over MongoDB**: Chosen for this service because feedback data (ratings, comments) is project-scoped and benefits from local transactional integrity. The embedded database eliminates operational complexity while WAL mode ensures adequate concurrent read performance for public API endpoints.
- **UPDATE-then-INSERT Upsert**: Reviews use an explicit update-then-insert pattern (rather than `INSERT OR REPLACE`) to preserve the original ULID and creation timestamp when a user updates their existing review.
- **Soft Delete for Comments**: Deleted comments are marked `status = 'deleted'` rather than physically removed, preserving reply thread integrity. Only the comment author can delete their own comment.
- **Auth Middleware vs Inline**: Auth is implemented as Gin middleware (applied per-route), an improvement over interfacer-dpp's inline handler pattern. `GET` and `OPTIONS` requests skip authentication entirely.
- **No Nested User Objects**: All responses return only `user_ulid` — the frontend is responsible for hydrating user metadata from the Identity service. Contract tests enforce this separation.
- **Zenroom Mutex**: All Zenroom cryptographic calls are serialized through a `sync.Mutex` because the `zencode-exec` library is not thread-safe.

## 📋 Prerequisites

Before running this application, ensure you have the following installed:

- **Go**: Version 1.25.0 or higher
- **zencode-exec**: Zenroom cryptographic VM binary (required for EdDSA signature verification)
  - Available from [Zenroom.org](https://zenroom.org/) or via the Dyne.org package repositories
- **SQLite**: Version 3.x (bundled via `modernc.org/sqlite`, no system-level install required)
- **Docker** (optional): For containerized deployment

## 🚀 Installation

### Local Development Setup

1. **Clone the repository with submodules**:
   ```bash
   git clone --recurse-submodules https://github.com/interfacerproject/interfacer-feedback-service.git
   cd interfacer-feedback-service
   ```

2. **Install Go dependencies**:
   ```bash
   go mod download
   ```

3. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your settings if needed
   ```

4. **Ensure zencode-exec is available**:
   ```bash
   which zencode-exec
   # If not found, install from https://zenroom.org/
   ```

5. **Run the application**:
   ```bash
   go run cmd/main/main.go
   ```

The API will be available at `http://localhost:8081`.

> ⚠️ **Note**: The application expects `.env` two directories above the binary (`../../.env`), matching the `cmd/main/` layout. When running from the project root, this resolves correctly.

## ⚙️ Configuration

### Environment Variables

The application is configured through environment variables (loaded from `.env` via `godotenv`). Copy `.env.example` to `.env` and adjust as needed:

| Variable | Default | Description |
|----------|---------|-------------|
| `SQLITE_PATH` | `./data/feedback.db` | Path to SQLite database file |
| `SERVER_URL` | `http://localhost:8081` | Public base URL of this service |
| `BASE_DID_URL` | `https://did.dyne.org` | Base URL for DID resolution (appended to public key) |
| `DID_CONTEXT_PATH` | `/path/to/did-context.json` | DID context path (e.g., `did:dyne:ifacer`) |

### Database Setup

The application automatically creates the SQLite database file, tables, and indexes on first run. No manual database setup or migration tooling is required.

The database is created with:
- **WAL mode** (`PRAGMA journal_mode=WAL`) for concurrent read performance
- **Foreign keys enabled** (`PRAGMA foreign_keys=ON`) for referential integrity
- **All tables and indexes** created via `IF NOT EXISTS` idempotent migrations

### Port Configuration

The service listens on port **8081** by default. To change this, modify the `router.Run()` call in `cmd/main/main.go` or set the `PORT` environment variable.

## 🔐 Authentication

This service uses the same DID-based EdDSA authentication flow as **interfacer-dpp**. The frontend/proxy sends signed requests, and the service verifies the signature using Zenroom cryptographic VM and DID resolution.

### Auth Headers

| Header | Description |
|--------|-------------|
| `did-sign` | EdDSA signature of the request body (base64-encoded) |
| `did-pk` | EdDSA public key of the signer |
| `x-user-id` | ULID of the authenticated user (set by proxy after successful auth) |

### Verification Flow

1. **Header Extraction**: Extract `did-sign`, `did-pk`, and `x-user-id` from request headers
2. **DID Resolution**: Fetch the DID document from `BASE_DID_URL` using the public key — returns 401 if the DID is not found or unreachable
3. **Zenroom Signature Verification**: Call `zencode-exec` with the embedded `verify_graphql.zen` script, passing the base64-encoded request body, the signature, and the public key — returns 401 on failure
4. **Context Propagation**: On success, `user_ulid` is set in the Gin context for downstream handlers

### Authorization Rules

| Endpoint Type | Auth Required | Authorization Rule |
|---------------|:---:|---|
| `GET` (all) | No | Public read access |
| `POST /reviews` | Yes | Any authenticated user; `UNIQUE(project_ulid, user_ulid)` limits one review per project |
| `DELETE /reviews/:id` | Yes | Must match review's `user_ulid` (author-only); returns 403 on mismatch |
| `POST /comments` | Yes | Any authenticated user |
| `DELETE /comments/:id` | Yes | Must match comment's `user_ulid` (author-only); returns 403 on mismatch |

### CORS Configuration

```
AllowAllOrigins:  true
AllowMethods:     GET, POST, PUT, DELETE, OPTIONS
AllowHeaders:     Origin, Content-Type, Accept, did-sign, did-pk, x-user-id
ExposeHeaders:    Content-Length
AllowCredentials: false
MaxAge:           12 hours
```

## 📚 API Documentation

### Base URL
```
http://localhost:8081
```

### Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| `POST` | `/api/v1/projects/{project_ulid}/reviews` | Yes | Create or update a review |
| `GET` | `/api/v1/projects/{project_ulid}/reviews` | No | List reviews (paginated) |
| `GET` | `/api/v1/projects/{project_ulid}/reviews/summary` | No | Get aggregated review stats |
| `GET` | `/api/v1/projects/{project_ulid}/reviews/mine` | No* | Get the current user's review (null if none) |
| `POST` | `/api/v1/projects/{project_ulid}/comments` | Yes | Create a comment or reply |
| `GET` | `/api/v1/projects/{project_ulid}/comments` | No | List comments (paginated, threaded) |
| `DELETE` | `/api/v1/reviews/{review_id}` | Yes | Delete a review (author-only) |
| `DELETE` | `/api/v1/comments/{comment_id}` | Yes | Soft-delete a comment (author-only) |

### GET /reviews — Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Maximum results per page (1–100) |
| `cursor` | integer | — | Unix timestamp of last item from previous page (for cursor pagination) |

Response:
```json
{
  "reviews": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAB",
      "user_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAC",
      "rating": 4,
      "content": "Great project with excellent documentation!",
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:30:00Z"
    }
  ]
}
```

### GET /reviews/summary — Response

```json
{
  "average_rating": 4.33,
  "total_reviews": 12,
  "rating_distribution": {
    "1": 0,
    "2": 1,
    "3": 2,
    "4": 4,
    "5": 5
  }
}
```

### GET /reviews/mine — Response

Returns the authenticated user's review for the project, or `null` if none exists.

Requires the `x-user-id` header. If not present, returns `{"review": null}`.

Response (review exists):
```json
{
  "review": {
    "id": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
    "project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAB",
    "user_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAC",
    "rating": 4,
    "content": "Great project with excellent documentation!",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
}
```

Response (no review):
```json
{
  "review": null
}
```

### DELETE /reviews/{review_id}

Permanently deletes a review. Only the review author can delete their own review.

Auth headers (`did-sign`, `did-pk`, `x-user-id`) are required.

Response:
```json
{
  "status": "deleted"
}
```

Error (not the author):
```json
{
  "error": "review not found or not owned by user"
}
```

### GET /comments — Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | integer | 20 | Maximum results per page (1–100) |
| `cursor` | integer | — | Unix timestamp of last item from previous page |
| `parent_id` | string | — | Fetch replies to a specific parent comment (omit for root comments) |

Response:
```json
{
  "comments": [
    {
      "id": "01ARZ3NDEKTSV4RRFFQ69G5FAX",
      "project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAB",
      "user_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAC",
      "parent_id": null,
      "content": "Has anyone tested this with the latest firmware?",
      "attachments": null,
      "status": "active",
      "created_at": "2025-02-01T14:00:00Z",
      "updated_at": "2025-02-01T14:00:00Z"
    }
  ]
}
```

### Error Response Format

All error responses follow the interfacer-dpp convention:

```json
{
  "error": "Human-readable error message",
  "details": "Optional detailed error information"
}
```

### HTTP Status Codes

| Status | Meaning |
|--------|---------|
| `200 OK` | Successful GET, PUT, DELETE operations |
| `201 Created` | Successful POST operation |
| `400 Bad Request` | Invalid request format, missing fields, or rating out of range |
| `401 Unauthorized` | Missing or invalid authentication headers |
| `403 Forbidden` | Attempted to delete another user's review or comment |
| `404 Not Found` | Resource not found |
| `500 Internal Server Error` | Server-side error |

## 📊 Data Model

### Review

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | TEXT (ULID) | PRIMARY KEY | Unique review identifier |
| `project_ulid` | TEXT | NOT NULL, INDEXED | Target project ULID |
| `user_ulid` | TEXT | NOT NULL, INDEXED | Review author ULID |
| `rating` | INTEGER | NOT NULL, CHECK 1–5 | Star rating |
| `content` | TEXT | NULLABLE | Optional review text |
| `created_at` | INTEGER | NOT NULL | Unix timestamp |
| `updated_at` | INTEGER | NOT NULL | Unix timestamp |

**Unique Constraint**: `UNIQUE(project_ulid, user_ulid)` — ensures one review per user per project. Subsequent `POST`s update the existing review.

### ReviewSummary (computed)

| Field | Type | Description |
|-------|------|-------------|
| `average_rating` | FLOAT64 | Mean of all ratings for the project |
| `total_reviews` | INTEGER | Total number of reviews |
| `rating_distribution` | OBJECT (int→int) | Count of reviews per rating value (1–5) |

### Comment

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `id` | TEXT (ULID) | PRIMARY KEY | Unique comment identifier |
| `project_ulid` | TEXT | NOT NULL, INDEXED | Target project ULID |
| `user_ulid` | TEXT | NOT NULL, INDEXED | Comment author ULID |
| `parent_id` | TEXT (ULID) | NULLABLE, INDEXED | Parent comment ULID (null = root comment) |
| `content` | TEXT | NOT NULL | Comment body text |
| `attachments` | TEXT (JSON) | NULLABLE | JSON array of file IDs/URLs from storage service |
| `status` | TEXT | DEFAULT 'active' | Comment status: `active` or `deleted` |
| `created_at` | INTEGER | NOT NULL | Unix timestamp |
| `updated_at` | INTEGER | NOT NULL | Unix timestamp |

### Database Indexes

```
idx_reviews_project_user   UNIQUE (project_ulid, user_ulid)
idx_comments_project        (project_ulid)
idx_comments_user           (user_ulid)
idx_comments_parent         (parent_id)
idx_comments_created        (created_at)
```

## 💡 Usage Examples

### Create a Review

```bash
curl -X POST http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/reviews \
  -H "Content-Type: application/json" \
  -H "did-sign: <base64-eddsa-signature>" \
  -H "did-pk: <eddsa-public-key>" \
  -H "x-user-id: 01ARZ3NDEKTSV4RRFFQ69G5FAC" \
  -d '{
    "rating": 5,
    "content": "Excellent project! The documentation is thorough and the API is intuitive."
  }'
```

### Update an Existing Review (same user, same project)

```bash
curl -X POST http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/reviews \
  -H "Content-Type: application/json" \
  -H "did-sign: <base64-eddsa-signature>" \
  -H "did-pk: <eddsa-public-key>" \
  -H "x-user-id: 01ARZ3NDEKTSV4RRFFQ69G5FAC" \
  -d '{
    "rating": 4,
    "content": "Updated my review after more testing."
  }'
```

### List Reviews (with pagination)

```bash
# First page
curl -X GET "http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/reviews?limit=10"

# Next page using cursor (created_at from last item)
curl -X GET "http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/reviews?limit=10&cursor=1737000000"
```

### Get Review Summary

```bash
curl -X GET http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/reviews/summary
```

### Post a Comment

```bash
curl -X POST http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/comments \
  -H "Content-Type: application/json" \
  -H "did-sign: <base64-eddsa-signature>" \
  -H "did-pk: <eddsa-public-key>" \
  -H "x-user-id: 01ARZ3NDEKTSV4RRFFQ69G5FAC" \
  -d '{
    "content": "Has anyone tested the material compatibility?",
    "attachments": "[\"file-ulid-1\", \"file-ulid-2\"]"
  }'
```

### Reply to a Comment

```bash
curl -X POST http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/comments \
  -H "Content-Type: application/json" \
  -H "did-sign: <base64-eddsa-signature>" \
  -H "did-pk: <eddsa-public-key>" \
  -H "x-user-id: 01ARZ3NDEKTSV4RRFFQ69G5FAF" \
  -d '{
    "parent_id": "01ARZ3NDEKTSV4RRFFQ69G5FAX",
    "content": "Yes, I tested with PLA and PETG — works great with both!"
  }'
```

### List Thread Replies

```bash
# Root-level comments
curl -X GET "http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/comments?limit=20"

# Replies to a specific comment
curl -X GET "http://localhost:8081/api/v1/projects/01ARZ3NDEKTSV4RRFFQ69G5FAB/comments?parent_id=01ARZ3NDEKTSV4RRFFQ69G5FAX&limit=20"
```

### Delete a Comment (author-only)

```bash
curl -X DELETE http://localhost:8081/api/v1/comments/01ARZ3NDEKTSV4RRFFQ69G5FAX \
  -H "did-sign: <base64-eddsa-signature>" \
  -H "did-pk: <eddsa-public-key>" \
  -H "x-user-id: 01ARZ3NDEKTSV4RRFFQ69G5FAC"
```

## 📡 Events

The service publishes asynchronous events to enable inter-service communication. When a review is created or updated, a `RatingUpdated` event is emitted so the main Project Service can update its cached aggregated ratings.

### RatingUpdated Event Schema

```json
{
  "event_type": "RatingUpdated",
  "project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAB",
  "average_rating": 4.33,
  "total_reviews": 12,
  "timestamp": 1737000000
}
```

### Publisher Interface

```go
type Publisher interface {
    PublishRatingUpdated(event RatingUpdatedEvent) error
}
```

The current implementation uses an in-process `LogPublisher` that logs events. To connect to a message broker (RabbitMQ, NATS, Kafka), implement the `Publisher` interface and instantiate it in `cmd/main/main.go`:

```go
// Replace the LogPublisher with a real broker implementation:
publisher := &broker.RabbitMQPublisher{URL: os.Getenv("BROKER_URL")}
```

Configure the broker URL in your `.env`:
```env
BROKER_URL=amqp://localhost:5672
```

### Event Flow

```
POST /reviews
     │
     ▼
UpsertReview (SQLite)
     │
     ▼ (goroutine)
GetReviewSummary (SQLite)
     │
     ▼
Publisher.PublishRatingUpdated()
     │
     ▼
Message Broker → Project Service
```

Events are published in a goroutine after the HTTP response is sent, so they do not add latency to the API response.

## 🛠️ Development

### Project Structure

The project follows Go best practices with a clean architecture aligned with interfacer-dpp conventions:

- **`cmd/`**: Application entry points
- **`internal/`**: Private application code
  - **`auth/`**: DID-based authentication (Zenroom + DID resolution)
  - **`database/`**: SQLite connection, migrations, and repository layer
  - **`handler/`**: HTTP request handlers (Gin controllers)
  - **`model/`**: Data structures and domain types
  - **`events/`**: Event publisher interface and event types

### Adding New Features

1. **Models**: Add new data structures in `internal/model/model.go`
2. **Repositories**: Implement data access in `internal/database/`
3. **Handlers**: Implement business logic in `internal/handler/`
4. **Routes**: Register new endpoints in `cmd/main/main.go`
5. **Auth**: If the endpoint requires authentication, apply `authMW` middleware

### Adding a New Event Type

1. Define the event struct in `internal/events/events.go`
2. Add a publish method to the `Publisher` interface
3. Implement the method in all publisher implementations
4. Call the publisher from the appropriate handler

### Code Style

- Follow standard Go formatting (`go fmt`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Handle errors appropriately — wrap with context using `fmt.Errorf("context: %w", err)`
- Use `*string` for nullable fields (matching interfacer-dpp convention)
- Preserve the `gin.H{"error": "...", "details": "..."}` error response format

### Updating the Zenroom Submodule

The `internal/auth/zenflows-crypto` directory is a git submodule pointing to the shared Zenroom scripts repository. To update it:

```bash
git submodule update --remote internal/auth/zenflows-crypto
```

## 🧪 Testing

The project includes unit tests, integration tests, and API contract tests.

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/database/...
go test ./internal/handler/...
```

### Test Structure

| Package | Test Type | Coverage |
|---------|-----------|----------|
| `internal/database` | Integration tests (in-memory SQLite) | Upsert logic, pagination, summary aggregation, soft delete, authorization rules |
| `internal/handler` | Contract tests | Response shape validation (no nested user objects), error response format, route registration |

### Contract Test Details

- **Response Shape**: Verifies that review and comment JSON responses contain only `user_ulid` and no nested `user`, `username`, `name`, `avatar`, `email`, or `profile` fields — enforcing the frontend hydration contract
- **Error Format**: Validates that error responses follow the `{"error": "...", "details": "..."}` DPP convention
- **Route Registration**: Basic sanity check that routes compile and return proper content types

## 🤝 Contributing

We welcome contributions to the Interfacer Feedback & Interaction Service! Please follow these guidelines:

1. **Fork the repository**
2. **Create a feature branch**: `git checkout -b feature/amazing-feature`
3. **Commit your changes**: `git commit -m 'Add amazing feature'`
4. **Push to the branch**: `git push origin feature/amazing-feature`
5. **Open a Pull Request**

### Contribution Guidelines

- Follow Go best practices and coding standards
- Add tests for new functionality
- Update documentation (this README) as needed
- Ensure all tests pass before submitting (`go test ./...`)
- Maintain response format conventions (`gin.H{"error": "..."}`)
- Keep the project structure aligned with interfacer-dpp conventions
- Use clear, descriptive commit messages

### Commit Format

Follow conventional commits where applicable:
```
feat: add event publishing for comment creation
fix: resolve cursor pagination edge case with zero timestamps
docs: update README with Docker deployment instructions
test: add integration tests for comment thread retrieval
```

---

## 😍 Acknowledgements

<a href="https://dyne.org">
  <img src="https://files.dyne.org/software_by_dyne.png" width="222">
</a>

Copyleft (ɔ) 2022–2025 by [Dyne.org](https://www.dyne.org) foundation, Amsterdam

Designed, written and maintained by the Interfacer Project team at Dyne.org.

**[🔝 back to top](#-table-of-contents)**

---

## 🌐 Links

https://www.interfacerproject.eu/

https://dyne.org/

**[🔝 back to top](#-table-of-contents)**

---

## 💼 License

    Interfacer Feedback & Interaction Service
    Copyleft (ɔ) 2022–2025 Dyne.org foundation

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU Affero General Public License as
    published by the Free Software Foundation, either version 3 of the
    License, or (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Affero General Public License for more details.

    You should have received a copy of the GNU Affero General Public License
    along with this program.  If not, see <http://www.gnu.org/licenses/>.
