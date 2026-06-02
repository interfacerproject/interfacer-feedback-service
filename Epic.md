# Epic: Feedback & Interaction Microservice (Golang + SQLite)

## 📖 Overview
This epic covers the end-to-end implementation of the Feedback Microservice. This service is responsible for handling user comments (with nested replies and file attachments) and project reviews/ratings.

It sits alongside **interfacer-dpp** (Digital Product Passport service) in the Interfacer Project ecosystem and shares its auth model, framework, and project conventions.

### Architectural Constraints & Assumptions
- **Language**: Go 1.25.0
- **Framework**: Gin HTTP web framework (matching interfacer-dpp)
- **Database**: SQLite (local, fast, self-contained)
- **Identifiers**: ULID (`github.com/oklog/ulid/v2`) — used for all internal primary keys and external references (`project_ulid`, `user_ulid`).
- **Auth Model**: DID-based EdDSA signature verification via Zenroom + DID resolution (identical flow to interfacer-dpp — see [Authentication](#-authentication) below).
- **User Identity**: The service extracts `user_ulid` from the `x-user-id` header set by the proxy/gateway after auth. Endpoints return only `user_ulid`; the frontend handles user metadata hydration from the Identity service.
- **Data Retention**: Users cannot delete their accounts, so no cascading deletion logic or anonymization processes are required for orphaned records.
- **Inter-service Communication**: The service will publish events (e.g., `ProjectRatingUpdated`) so the main Project Service can update its aggregated average ratings.

### Project Structure (aligned with interfacer-dpp)
```
interfacer-feedback-service/
├── cmd/
│   └── main/
│       └── main.go           # Application entry point, route registration
├── internal/
│   ├── auth/
│   │   └── auth.go           # DID verification (Zenroom + DID resolution)
│   ├── database/
│   │   └── database.go       # SQLite connection, PRAGMAs, migrations
│   ├── handler/
│   │   ├── reviews.go        # Review endpoints
│   │   └── comments.go       # Comment endpoints
│   └── model/
│       └── model.go          # Data structures: Review, Comment
├── .env.example              # Environment variable template
├── go.mod
├── go.sum
└── README.md
```

---

## 🔐 Authentication

This service uses the same DID-based EdDSA authentication flow as **interfacer-dpp**. The frontend/proxy sends signed requests, and the service verifies the signature using Zenroom cryptographic VM and DID resolution.

### Auth Headers

| Header       | Description |
|-------------|-------------|
| `did-sign`  | EdDSA signature of the request body (base64-encoded) |
| `did-pk`    | EdDSA public key of the signer |
| `x-user-id` | ULID of the authenticated user (set by proxy after successful auth) |

### Verification Flow (mirrored from interfacer-dpp)

1. Extract `did-sign`, `did-pk`, and `x-user-id` from request headers.
2. **DID Resolution**: Fetch the DID document from `BASE_DID_URL` (env var) using the public key. Returns 401 if the DID is not found or unreachable.
3. **Zenroom Signature Verification**: Call `zencode-exec` with the embedded `verify_graphql.zen` script, passing the base64-encoded request body (`gql`), the signature (`eddsa_signature`), and the public key (`eddsa_public_key`). Returns 401 on failure.
   - Note: Zenroom calls must be **serialized** via a `sync.Mutex` — the library is not thread-safe.
4. On success, the handler uses `x-user-id` as `user_ulid` for all database writes.

### Key Implementation Details

- The Zenroom verify script (`verify_graphql.zen`) is embedded via Go's `//go:embed` directive.
- `zencode-exec` is called via `exec.Command` with proper pipe handling to avoid deadlock on large inputs (>4KB pipe buffer).
- The auth package exposes `ZenroomData.VerifyDid()` and `ZenroomData.IsAuth()` methods identical to interfacer-dpp.
- CORS must allow `did-sign`, `did-pk`, and `x-user-id` headers.

### Authorization (per-endpoint)

- **Reviews**: Any authenticated user can create/update their own review. The `UNIQUE(project_ulid, user_ulid)` constraint ensures one review per user per project.
- **Comments**: Any authenticated user can post comments. Deletion is **author-only** — the `x-user-id` header must match the comment's `user_ulid`.
- **Read endpoints** (`GET`): No auth required (public access).

---

## 🗄️ Database Schema (SQLite)

### Table: `reviews`
Stores the 1-to-5 star ratings and optional review text.
- `id` (TEXT, Primary Key) - ULID generated on creation.
- `project_ulid` (TEXT, Not Null, Indexed)
- `user_ulid` (TEXT, Not Null, Indexed)
- `rating` (INTEGER, Not Null) - Check constraint: 1 to 5.
- `content` (TEXT, Nullable)
- `created_at` (INTEGER, Not Null) - Unix timestamp.
- `updated_at` (INTEGER, Not Null) - Unix timestamp.
- **Constraint**: `UNIQUE(project_ulid, user_ulid)`

### Table: `comments`
Stores hierarchical comments and discussions.
- `id` (TEXT, Primary Key) - ULID generated on creation.
- `project_ulid` (TEXT, Not Null, Indexed)
- `user_ulid` (TEXT, Not Null, Indexed)
- `parent_id` (TEXT, Nullable, Indexed) - ULID of the parent comment. Null if root comment.
- `content` (TEXT, Not Null)
- `attachments` (TEXT, Nullable) - JSON array of file IDs/URLs from the Storage Service.
- `status` (TEXT, Default 'active') - e.g., 'active', 'deleted' (soft delete).
- `created_at` (INTEGER, Not Null) - Unix timestamp.
- `updated_at` (INTEGER, Not Null) - Unix timestamp.

---

## 🛤️ Implementation Plan (Tasks)

### Phase 0: Auth Middleware (shared with interfacer-dpp pattern)
- [ ] **0.1 Copy/port auth package from interfacer-dpp**
  - Copy `internal/auth/auth.go` and the embedded `zenflows-crypto/src/verify_graphql.zen` from interfacer-dpp.
  - Adjust Go module path. No logic changes needed — the signature verification flow is identical.
  - **Dependency**: requires `zencode-exec` binary on the host (same as DPP).
- [ ] **0.2 Create Gin auth middleware**
  - Extract `did-sign`, `did-pk`, `x-user-id` from headers.
  - Perform DID resolution + Zenroom signature verification.
  - On failure: abort with 401 + JSON error (matching DPP's `{"error":"Authentication failed","details":"..."}` format).
  - On success: set `user_ulid` in Gin context for downstream handlers.
  - Skip auth for `GET` endpoints (public reads) and `OPTIONS` (CORS preflight).

### Phase 1: Project Setup & Database Boilerplate
- [ ] **1.1 Initialize Go Module**
  - Create `go.mod` with module path `github.com/interfacerproject/interfacer-feedback-service`.
  - Create project structure: `cmd/main/`, `internal/{auth,database,handler,model}` (matching DPP layout).
- [ ] **1.2 Setup Dependencies**
  - **Gin**: `github.com/gin-gonic/gin` (matching DPP)
  - **CORS**: `github.com/gin-contrib/cors` (matching DPP)
  - **SQLite**: `github.com/mattn/go-sqlite3` or pure Go `modernc.org/sqlite`
  - **ULID**: `github.com/oklog/ulid/v2` (same as DPP)
  - **Dotenv**: `github.com/joho/godotenv` (matching DPP)
- [ ] **1.3 Database Connection & Migrations**
  - Implement SQLite connection initialization with a `sync.Once` singleton pattern (matching DPP's `ConnectDB`).
  - Auto-create `reviews` and `comments` tables on startup with correct PRAGMAs: `PRAGMA foreign_keys = ON; PRAGMA journal_mode=WAL;`.
  - Create indexes: `project_ulid`, `user_ulid`, `parent_id`, `created_at`.

### Phase 2: Data Access Layer (Repositories)
- [ ] **2.1 Implement `ReviewRepository`**
  - `UpsertReview(projectULID, userULID, rating, content)`: Uses SQLite `INSERT ... ON CONFLICT(project_ulid, user_ulid) DO UPDATE` to ensure users can only leave one review, but can update it.
  - `GetReviewsByProject(projectULID, limit, cursor)`: Fetch paginated reviews.
  - `GetReviewSummary(projectULID)`: Execute SQL aggregation to return `average_rating`, `total_reviews`, and `rating_distribution` (count of 1s, 2s, 3s, etc.).
- [ ] **2.2 Implement `CommentRepository`**
  - `InsertComment(projectULID, userULID, parentID, content, attachments)`: Insert new comment.
  - `GetCommentsByProject(projectULID, parentID, limit, cursor)`: Fetch comments. If `parentID` is null, fetch root comments.
  - `SoftDeleteComment(commentID, userULID)`: Update `status = 'deleted'` to keep the thread hierarchy intact without showing content.

### Phase 3: REST API - Reviews
- [ ] **3.1 `POST /api/v1/projects/{project_ulid}/reviews`** (auth required)
  - Validate payload: rating must be 1-5.
  - Extract `user_ulid` from Gin context (set by auth middleware).
  - Call `UpsertReview`.
  - **Crucial**: Trigger asynchronous event `ProjectRatingUpdated` (see Phase 5).
- [ ] **3.2 `GET /api/v1/projects/{project_ulid}/reviews`** (public)
  - Return paginated list of reviews.
  - Response structure includes `user_ulid` (Frontend will handle hydration).
- [ ] **3.3 `GET /api/v1/projects/{project_ulid}/reviews/summary`** (public)
  - Return aggregated stats: average rating and total count.

### Phase 4: REST API - Comments
- [ ] **4.1 `POST /api/v1/projects/{project_ulid}/comments`** (auth required)
  - Validate payload (content must not be empty).
  - Extract `user_ulid` from Gin context.
  - Accept optional `parent_id` (for replies) and `attachments` (array of strings from File Storage Service).
- [ ] **4.2 `GET /api/v1/projects/{project_ulid}/comments`** (public)
  - Accept query params: `cursor` (for pagination) and `parent_id` (to fetch replies of a specific thread).
  - Return comments ordered by `created_at`.
- [ ] **4.3 `DELETE /api/v1/comments/{comment_id}`** (auth required)
  - Soft-delete logic. Ensure only the author can delete: compare `x-user-id` header with the comment's `user_ulid`. Return 403 on mismatch.

### Phase 5: Event Emitting (Async Communication)
- [ ] **5.1 Setup Event Publisher**
  - Integrate with the chosen Message Broker (e.g., RabbitMQ, NATS, Kafka) or a lightweight Pub/Sub.
  - Define the event payload schema:
    ```json
    {
      "event_type": "RatingUpdated",
      "project_ulid": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
      "average_rating": 4.5,
      "total_reviews": 12,
      "timestamp": 1672531199
    }
    ```
- [ ] **5.2 Emit Event on Review Upsert**
  - After a successful `POST /reviews`, recalculate the summary using `GetReviewSummary`.
  - Publish the `RatingUpdated` event so the external **Project Service** can cache the new average on its side.

### Phase 6: Testing & QA
- [ ] **6.1 Unit Tests**
  - Write table-driven tests for Go handlers and ULID generation logic.
  - Test auth middleware: valid signature, invalid signature, missing headers.
- [ ] **6.2 Integration Tests (SQLite In-Memory)**
  - Spin up an in-memory SQLite DB (`file::memory:?cache=shared`) to test Repository SQL queries.
  - Test the `ON CONFLICT` logic for reviews.
  - Test pagination mechanisms.
- [ ] **6.3 API Contract Verification**
  - Ensure all JSON responses accurately return `user_ulid` without attempting to return nested user objects, strictly adhering to the Frontend-hydration contract.
  - Verify error response format matches DPP conventions: `{"error": "message", "details": "..."}`.

---

## 🔗 Alignment with interfacer-dpp

| Concern | interfacer-dpp | This Service |
|--------|---------------|-------------|
| Language | Go 1.24.0 | Go 1.25.0 |
| Framework | Gin | Gin |
| CORS | `AllowAllOrigins: true`, headers: `did-sign`, `did-pk`, `x-user-id` | Same |
| Auth | DID + Zenroom EdDSA verification | Same (ported auth package) |
| ID type | `github.com/oklog/ulid/v2` | Same |
| Env loading | `godotenv.Load("../../.env")` | Same |
| DB singleton | `sync.Once` in `ConnectDB()` | Same pattern |
| Project layout | `cmd/main/`, `internal/{auth,database,handler,model,storage}` | `cmd/main/`, `internal/{auth,database,handler,model}` |
| Error format | `{"error":"...", "details":"..."}` | Same |
| Auth on writes | Inline in handler | Middleware (improvement) |
| File storage | MinIO (S3-compatible) | Not needed in v1 (attachment URLs only) |
