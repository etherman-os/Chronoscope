# Go Security Audit Report

**Scope**: `services/ingestion/` and `services/analytics/`  
**Date**: 2026-04-25  
**Auditor**: Security & Quality Auditor (Go)

---

## CRITICAL Findings

### [C-001] Horizontal Authorization Bypass in `ListSessions`
- **File**: `services/ingestion/internal/handlers/session.go`
- **Line**: 87–95
- **Issue**: The handler accepts an arbitrary `project_id` query parameter and uses it directly in the SQL query without validating that it matches the authenticated project (from the API key middleware). An attacker with a valid API key for Project A can exfiltrate sessions from Project B by passing `?project_id=<project_b_id>`.
- **Impact**: Complete data breach across all projects. Any authenticated client can read any other project's session data.
- **Fix**: Remove the `project_id` query parameter entirely and always use `c.Get("project_id")` from the auth context, or strictly validate that the query parameter matches the authenticated project ID.
  ```go
  // FIX: Always use the authenticated project ID
  projectID, ok := c.Get("project_id")
  if !ok {
      c.JSON(http.StatusUnauthorized, gin.H{"error": "missing project context"})
      return
  }
  ```

### [C-002] Horizontal Authorization Bypass on Session Mutations
- **Files**:
  - `services/ingestion/internal/handlers/events.go` (line 26)
  - `services/ingestion/internal/handlers/chunk.go` (line 16)
  - `services/ingestion/internal/handlers/complete.go` (line 13)
  - `services/ingestion/internal/handlers/session.go` (line 136, GetSession)
- **Issue**: None of these handlers verify that the requested `sessionID` belongs to the project associated with the authenticated API key. An attacker with a valid key for Project A can upload events, upload chunks, complete, or read sessions belonging to Project B if they know or can guess the session UUID.
- **Impact**: Data pollution, unauthorized session completion, and information disclosure across tenant boundaries.
- **Fix**: Before mutating or returning a session, verify ownership:
  ```go
  var ownerProjectID string
  err := cfg.DB.QueryRow(`SELECT project_id FROM sessions WHERE id = $1`, sessionID).Scan(&ownerProjectID)
  if err != nil || ownerProjectID != authenticatedProjectID {
      c.JSON(http.StatusForbidden, gin.H{"error": "session does not belong to project"})
      return
  }
  ```

---

## HIGH Findings

### [H-001] API Key Stored/Compared in Plaintext
- **Files**:
  - `services/ingestion/internal/middleware/auth.go` (line 20)
  - `services/analytics/internal/middleware/auth.go` (line 20)
- **Issue**: The database column is named `api_key_hash`, yet the middleware performs a direct equality comparison between the raw header value (`apiKey`) and the column. This strongly implies either (a) the column stores plaintext API keys despite its name, or (b) the developer forgot to hash the incoming key before comparison.
- **Impact**: If the database is compromised, attacker obtains usable API keys immediately. Also violates the principle of least surprise and secure naming conventions.
- **Fix**: Hash the incoming API key (e.g., with SHA-256 or bcrypt) before querying, or rename the column to `api_key` if plaintext storage is intentional (not recommended).
  ```go
  // Example with SHA-256
  h := sha256.Sum256([]byte(apiKey))
  hashHex := hex.EncodeToString(h[:])
  err := db.QueryRow("SELECT id FROM projects WHERE api_key_hash = $1", hashHex).Scan(&projectID)
  ```

### [H-002] CORS Allow-Origin Set to `*` (Wildcard)
- **Files**:
  - `services/ingestion/internal/middleware/cors.go` (line 12)
  - `services/analytics/internal/middleware/cors.go` (line 12)
- **Issue**: Both services return `Access-Control-Allow-Origin: *`, allowing any website to make cross-origin requests to the APIs. In a production environment with API-key-based authentication, this increases the attack surface for CSRF-like attacks and credential stuffing from arbitrary origins.
- **Impact**: Any malicious website can invoke these APIs from a victim's browser, facilitating cross-origin attacks.
- **Fix**: Restrict origins via an environment variable:
  ```go
  allowedOrigin := os.Getenv("CORS_ALLOWED_ORIGIN")
  if allowedOrigin == "" {
      allowedOrigin = "https://app.chronoscope.io"
  }
  c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
  ```

### [H-003] Unvalidated File Upload — No Size or Content Checks
- **File**: `services/ingestion/internal/handlers/chunk.go`
- **Lines**: 29–44
- **Issue**: The chunk upload handler accepts any multipart file without enforcing:
  - Maximum file size
  - Actual content-type verification (the code hardcodes `image/jpeg` without inspecting the file magic bytes)
  - File extension validation
  - Maximum chunk index or total upload size
  The `PutObject` call uses `-1` as the size, meaning it will stream until EOF.
- **Impact**: Denial of Service via disk/memory exhaustion; potential storage abuse; clients can upload arbitrary file types disguised as JPEG chunks.
- **Fix**:
  ```go
  const maxChunkSize = 2 << 20 // 2 MiB
  c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxChunkSize)
  file, header, err := c.Request.FormFile("chunk")
  if err != nil { /* handle */ }
  if header.Size > maxChunkSize {
      c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "chunk too large"})
      return
  }
  // Optionally inspect first bytes for JPEG magic numbers
  ```

### [H-004] Audit Log Errors Silently Ignored
- **Files** (all occurrences use `_ = LogAudit(...)`):
  - `services/ingestion/internal/handlers/session.go` (line 69)
  - `services/ingestion/internal/handlers/events.go` (line 87)
  - `services/ingestion/internal/handlers/chunk.go` (line 48)
  - `services/ingestion/internal/handlers/complete.go` (line 26)
  - `services/ingestion/internal/handlers/gdpr.go` (line 179)
- **Issue**: Every call to `LogAudit` discards the returned error. If audit logging fails (e.g., DB write error), the application continues as if the audit trail was successfully recorded. This breaks compliance guarantees and non-repudiation.
- **Impact**: Regulatory/compliance violations; inability to reconstruct security incidents.
- **Fix**: At minimum log the error; ideally return a 500 if audit logging is critical:
  ```go
  if err := LogAudit(cfg, pid, "session_initiated", req.UserID, details); err != nil {
      log.Printf("audit log failed: %v", err)
  }
  ```

### [H-005] No Rate Limiting
- **Files**:
  - `services/ingestion/cmd/server/main.go`
  - `services/analytics/cmd/server/main.go`
- **Issue**: Neither service implements any form of rate limiting. Brute-force attacks against API keys, mass session creation, and large event-batch submissions are unrestricted.
- **Impact**: Account takeover via brute force, resource exhaustion, and abuse of ingestion endpoints.
- **Fix**: Introduce a middleware-based rate limiter (e.g., using `golang.org/x/time/rate` or a Redis-backed counter) keyed by API key or client IP.

---

## MEDIUM Findings

### [M-001] No Request Body Size Limits
- **Files**:
  - `services/ingestion/cmd/server/main.go`
  - `services/analytics/cmd/server/main.go`
- **Issue**: Neither Gin engine nor HTTP server sets a maximum request body size. Large JSON payloads (e.g., massive event batches) can exhaust server memory.
- **Impact**: Denial of Service via memory exhaustion.
- **Fix**: Set `router.MaxMultipartMemory` and use `http.MaxBytesReader` on request bodies.

### [M-002] Missing Database Query Timeouts
- **Files**: All handler files across both services
- **Issue**: Every `db.Query`, `db.QueryRow`, and `db.Exec` call uses the default background context with no timeout. Slow queries can accumulate and exhaust the connection pool.
- **Impact**: Cascade failures under load; potential connection pool exhaustion.
- **Fix**: Use `context.WithTimeout` for each request:
  ```go
  ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
  defer cancel()
  rows, err := cfg.DB.QueryContext(ctx, `...`, args...)
  ```

### [M-003] Hardcoded MinIO `Secure: false`
- **File**: `services/ingestion/internal/config/config.go`
- **Line**: 53
- **Issue**: The MinIO client is initialized with `Secure: false` unconditionally. If the MinIO endpoint is accessed over an untrusted network, credentials and data are transmitted in plaintext.
- **Impact**: Man-in-the-middle attacks; credential theft.
- **Fix**: Drive the flag from an environment variable:
  ```go
  secure := os.Getenv("MINIO_SECURE") == "true"
  ```

### [M-004] Inconsistent Error Handling on Row Scan Failures
- **Files**:
  - `services/ingestion/internal/handlers/session.go` (lines 124–125)
  - `services/ingestion/internal/handlers/gdpr.go` (lines 37, 61)
- **Issue**: Some handlers silently `continue` on `rows.Scan` errors, dropping corrupted records without logging. Others (e.g., `analytics/handlers/heatmap.go`) abort the entire request.
- **Impact**: Silent data loss or inconsistent API behavior.
- **Fix**: Adopt a single pattern—log the error and decide whether to skip or abort based on business rules.

### [M-005] No Maximum Event Batch Size
- **File**: `services/ingestion/internal/handlers/events.go`
- **Line**: 55–68
- **Issue**: The `UploadEvents` handler iterates over `req.Events` with no upper bound. A malicious or buggy client can send an enormous array, causing memory pressure and a very long-running transaction.
- **Impact**: Memory exhaustion; long-lived transactions holding locks.
- **Fix**: Enforce a maximum batch size (e.g., 1,000 events) before entering the loop.

### [M-006] No Maximum Chunk Index Validation
- **File**: `services/ingestion/internal/handlers/chunk.go`
- **Line**: 23–27
- **Issue**: `chunkIndex` is validated as non-negative but has no upper bound. An attacker could upload `chunk_99999999.jpg`, exhausting object storage namespace and indexes.
- **Impact**: Storage pollution; potential performance degradation in MinIO.
- **Fix**: Enforce a reasonable maximum (e.g., `chunkIndex < 10000`).

### [M-007] Generated Session Token Never Validated
- **File**: `services/ingestion/internal/handlers/session.go`
- **Lines**: 72–73, 75–80
- **Issue**: `InitSession` generates a UUID token and returns it in the response, but there is no handler or middleware that validates this token on subsequent requests (chunks, events, complete).
- **Impact**: The token is security theater; it provides no protection against session hijacking if a session ID is guessed or leaked.
- **Fix**: Either remove the token (if unused) or implement token validation middleware for session-scoped endpoints.

### [M-008] Transaction/Storage Consistency Gap in `DeleteUserData`
- **File**: `services/ingestion/internal/handlers/gdpr.go`
- **Lines**: 132–176
- **Issue**: MinIO objects are deleted *outside* the database transaction. If the server crashes after MinIO deletions but before `tx.Commit()`, the database retains references to now-missing storage objects.
- **Impact**: Orphaned metadata; broken referential integrity.
- **Fix**: Delete MinIO objects only after the DB transaction commits successfully, or implement a two-phase cleanup / reconciliation job.

---

## LOW Findings

### [L-001] Dead Code: `storage/minio.go` Wrapper Unused
- **File**: `services/ingestion/internal/storage/minio.go`
- **Issue**: `MinioClient`, `NewMinioClient`, and `UploadObject` are defined but never referenced anywhere. Handlers use `cfg.Minio` (a raw `*minio.Client`) directly.
- **Impact**: Maintenance burden; confusion for future developers.
- **Fix**: Remove the unused package or migrate handlers to use the wrapper.

### [L-002] Ignored `RowsAffected` Error
- **File**: `services/ingestion/internal/handlers/gdpr.go`
- **Line**: 160
- **Issue**: `evCount, _ := res.RowsAffected()` discards the error. If the driver fails to report affected rows, the audit log will under-report deletions.
- **Impact**: Inaccurate audit metrics.
- **Fix**: Handle or log the error.

### [L-003] Duplicate CORS and Auth Middleware
- **Files**:
  - `services/ingestion/internal/middleware/cors.go` (identical to analytics)
  - `services/ingestion/internal/middleware/auth.go` (identical to analytics)
- **Issue**: The two services maintain exact copies of CORS and API-key auth middleware. Any security fix must be applied in two places, increasing the risk of drift.
- **Impact**: Maintenance overhead; risk of inconsistent security policies.
- **Fix**: Extract common middleware into a shared internal library (e.g., `github.com/chronoscope/pkg/middleware`).

### [L-004] Analytics Server Lacks Graceful Shutdown
- **File**: `services/analytics/cmd/server/main.go`
- **Line**: 26
- **Issue**: The analytics service calls `router.Run(":8081")` directly, unlike the ingestion service which uses `http.Server` with graceful shutdown on `SIGTERM`/`SIGINT`.
- **Impact**: In-flight requests may be abruptly terminated during deploys or scaling events.
- **Fix**: Mirror the ingestion service's shutdown pattern using `http.Server` and `srv.Shutdown`.

---

## Checklist Summary

| Check | Status | Notes |
|-------|--------|-------|
| All SQL queries use parameterized statements | ✅ PASS | All `$N` placeholders; no string concatenation |
| No `fmt.Sprintf` in SQL queries | ✅ PASS | Verified across all handler files |
| API key validated on every protected route | ✅ PASS | All `/v1/*` routes use `APIKeyAuth` middleware |
| No `panic()` in HTTP handlers | ✅ PASS | No panic/recover in handlers; startup `log.Fatal` is acceptable |
| CORS restricted to known origins | ❌ FAIL | `*` hardcoded in both services ([H-002]) |
| File upload validation | ❌ FAIL | No size/type checks ([H-003]) |
| No hardcoded secrets | ✅ PASS | Secrets sourced from env vars |
| Environment variables validated at startup | ✅ PASS | `DATABASE_URL`, MinIO vars checked |
| Request body size limits | ❌ FAIL | No explicit limits ([M-001]) |
| Rate limiting | ❌ FAIL | No rate limiting implemented ([H-005]) |

**Findings Count**:  
- CRITICAL: **2**  
- HIGH: **5**  
- MEDIUM: **8**  
- LOW: **4**
