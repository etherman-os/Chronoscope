# Go Quality Audit Report

**Scope**: `services/ingestion/` and `services/analytics/`  
**Date**: 2026-04-25  
**Auditor**: Security & Quality Auditor (Go)

---

## Lint Issues

### [LINT-001] Non-Canonical Import Ordering
- **Files**:
  - `services/analytics/internal/handlers/stats.go` (lines 3–9)
  - `services/analytics/internal/handlers/funnel.go` (lines 3–8)
  - `services/analytics/internal/handlers/heatmap.go` (lines 3–8)
  - `services/analytics/internal/config/config.go` (lines 3–9)
  - `services/analytics/cmd/server/main.go` (lines 3–10)
- **Issue**: Within the third-party import group, `github.com/chronoscope/analytics/...` internal packages are placed after `github.com/gin-gonic/gin`, violating `goimports` / `gofmt` alphabetical ordering (`c` comes before `g`). The ingestion service mostly follows correct ordering, but analytics is inconsistent.
- **Fix**: Run `goimports -w` across the analytics service.
  ```go
  // Before
  import (
      "github.com/gin-gonic/gin"
      "github.com/chronoscope/analytics/internal/config"
  )
  // After
  import (
      "github.com/chronoscope/analytics/internal/config"
      "github.com/gin-gonic/gin"
  )
  ```

### [LINT-002] Struct Tag Alignment
- **File**: `services/analytics/internal/handlers/stats.go`
- **Line**: 12–17
- **Issue**: `AvgEventsPerSession` tag is not aligned with the other fields, reducing readability.
- **Fix**: Align JSON tags with spaces.

---

## Dead Code

### [DEAD-001] Unused `storage` Package
- **File**: `services/ingestion/internal/storage/minio.go`
- **Issue**: The entire `storage` package (`MinioClient`, `NewMinioClient`, `UploadObject`) is compiled into the binary but never invoked. Handlers call `cfg.Minio.PutObject` directly using the raw `*minio.Client`.
- **Fix**: Delete the `storage` directory and remove the dead abstraction, or refactor handlers to consume the wrapper for consistency.

### [DEAD-002] Session Token Generated but Never Consumed
- **File**: `services/ingestion/internal/handlers/session.go`
- **Lines**: 72–73
- **Issue**: `token := uuid.New().String()` is created and returned in the JSON response, yet no endpoint validates it. This dead feature implies incomplete session-security logic.
- **Fix**: Implement token validation in chunk/event/complete handlers, or remove the field from the response to reduce API surface area.

---

## Style Issues

### [STYLE-001] Inconsistent Slice Initialization
- **Files**:
  - `services/ingestion/internal/handlers/session.go` (line 120)
  - `services/ingestion/internal/handlers/gdpr.go` (lines 31, 220)
- **Issue**: `sessions := []models.Session{}` and `logs := []map[string]interface{}{}` use literal initialization. The more idiomatic Go style for zero-length slices destined for appending is `var sessions []models.Session`.
- **Impact**: Nil vs. empty JSON array differences can surface in API responses. The current literal form is not wrong but inconsistent with Go idioms.
- **Fix**: Standardize on `var x []T` unless an empty literal is explicitly required.

### [STYLE-002] Duplicate Middleware Code Across Services
- **Files**:
  - `services/ingestion/internal/middleware/cors.go` ↔ `services/analytics/internal/middleware/cors.go`
  - `services/ingestion/internal/middleware/auth.go` ↔ `services/analytics/internal/middleware/auth.go`
- **Issue**: Both files are byte-for-byte identical. Copy-pasted code makes refactors error-prone and increases binary size slightly.
- **Fix**: Extract shared middleware into a `pkg/middleware` module or a `shared/` directory consumed by both services.

### [STYLE-003] Inconsistent Server Lifecycle Patterns
- **Files**:
  - `services/ingestion/cmd/server/main.go`
  - `services/analytics/cmd/server/main.go`
- **Issue**: Ingestion implements graceful shutdown with `http.Server` + signal handling. Analytics uses the shorthand `router.Run(":8081")`, which does not handle `SIGTERM` gracefully.
- **Fix**: Align analytics `main.go` with the ingestion pattern for consistency and operational reliability.

---

## Error Handling Issues

### [ERR-001] Silent Suppression of `rows.Scan` Errors
- **Files**:
  - `services/ingestion/internal/handlers/session.go` (line 124–125)
  - `services/ingestion/internal/handlers/gdpr.go` (lines 37, 61, 123–124, 229–230)
- **Issue**: On `rows.Scan` failure, the code executes `continue`, silently dropping the record. No logging is performed, making data-corruption or schema-mismatch bugs impossible to detect in production.
- **Fix**: Log the error at minimum:
  ```go
  if err := rows.Scan(&s.ID, ...); err != nil {
      log.Printf("scan session row: %v", err)
      continue
  }
  ```

### [ERR-002] Inconsistent Row-Scan Failure Strategies
- **Files**:
  - `services/ingestion/internal/handlers/session.go` (line 124): `continue`
  - `services/ingestion/internal/handlers/gdpr.go` (line 37): `continue`
  - `services/analytics/internal/handlers/heatmap.go` (lines 44–47): abort with 500
- **Issue**: There is no documented rule for when to skip a bad row versus failing the entire request. This inconsistency confuses API consumers and complicates debugging.
- **Fix**: Define a project-wide convention (e.g., "skip and log for list endpoints; abort for aggregate endpoints") and document it.

### [ERR-003] Audit Log Errors Ignored
- **Files** (all use `_ = LogAudit(...)`):
  - `services/ingestion/internal/handlers/session.go` (line 69)
  - `services/ingestion/internal/handlers/events.go` (line 87)
  - `services/ingestion/internal/handlers/chunk.go` (line 48)
  - `services/ingestion/internal/handlers/complete.go` (line 26)
  - `services/ingestion/internal/handlers/gdpr.go` (line 179)
- **Issue**: The error return from `LogAudit` is always discarded. If the audit table is unavailable or misconfigured, the application remains silent.
- **Fix**: Propagate the error to the application log:
  ```go
  if err := LogAudit(...); err != nil {
      log.Printf("audit log failed: %v", err)
  }
  ```

### [ERR-004] `RowsAffected` Error Discarded
- **File**: `services/ingestion/internal/handlers/gdpr.go`
- **Line**: 160
- **Code**:
  ```go
  evCount, _ := res.RowsAffected()
  ```
- **Issue**: The error from `RowsAffected` is ignored. Under rare driver failures, the deletion count reported to the client and stored in the audit log will be inaccurate.
- **Fix**: Log or handle the error.

### [ERR-005] Missing `eventRows` Defer Close
- **File**: `services/ingestion/internal/handlers/gdpr.go`
- **Lines**: 40–77
- **Issue**: `eventRows` is closed explicitly at line 77, but if an early return occurs (e.g., line 45–46), the close is skipped. While `eventRows.Close()` on a nil result set is safe, an explicit `defer eventRows.Close()` immediately after the successful query would be more robust.
- **Fix**:
  ```go
  eventRows, err := cfg.DB.Query(...)
  if err != nil { /* ... */ }
  defer eventRows.Close()
  ```

---

## Test Coverage Gaps

### [TEST-001] Zero Unit Tests in Ingestion Service
- **Path**: `services/ingestion/`
- **Finding**: No `*_test.go` files exist anywhere in the ingestion service tree.
- **Gaps**:
  - No tests for `InitSession`, `ListSessions`, `GetSession`
  - No tests for `UploadEvents` batch insertion and transaction rollback
  - No tests for `UploadChunk` multipart handling
  - No tests for `CompleteSession`
  - No tests for `ExportUserData` or `DeleteUserData`
  - No tests for `APIKeyAuth` middleware success/failure paths
  - No tests for `LogAudit`

### [TEST-002] Zero Unit Tests in Analytics Service
- **Path**: `services/analytics/`
- **Finding**: No `*_test.go` files exist anywhere in the analytics service tree.
- **Gaps**:
  - No tests for `GetSessionStats` aggregation logic
  - No tests for `GetFunnel` multi-query coordination
  - No tests for `GetHeatmap` grouping and limiting
  - No tests for `APIKeyAuth` middleware

### [TEST-003] No Integration Tests
- **Finding**: There are no tests that spin up a real or in-memory HTTP server, database, or MinIO instance to validate end-to-end behavior.
- **Recommendation**: Add integration tests using `testcontainers-go` or an in-memory SQLite/PostgreSQL harness to verify SQL correctness and handler wiring.

---

## Additional Quality Observations

### [Q-001] Magic Numbers
- **Files**:
  - `services/ingestion/internal/handlers/session.go` (line 73): `1 * time.Hour` (token expiry)
  - `services/ingestion/internal/handlers/gdpr.go` (line 197): `limit := 20`
  - `services/ingestion/internal/handlers/session.go` (line 97): `limit := 20`
- **Issue**: Hardcoded pagination defaults and TTLs make operational tuning impossible without recompiling.
- **Fix**: Extract to environment variables or a constants package.

### [Q-002] Missing Connection Pool Configuration
- **Files**:
  - `services/ingestion/internal/config/config.go`
  - `services/analytics/internal/config/config.go`
- **Issue**: `sql.Open` is called without setting `db.SetMaxOpenConns`, `db.SetMaxIdleConns`, or `db.SetConnMaxLifetime`. Under load, the services may exhaust the PostgreSQL connection pool or hold stale connections.
- **Fix**: Expose pool settings via environment variables and configure them after `sql.Open`.

### [Q-003] Missing `Content-Type` Validation on JSON Endpoints
- **Files**: All handlers using `c.ShouldBindJSON`
- **Issue**: Gin's `ShouldBindJSON` will attempt to bind even if the request `Content-Type` is `text/plain` or missing. While not strictly a bug, it is sloppy API behavior.
- **Fix**: Either rely on Gin's strict binding (`BindJSON`) which enforces `application/json`, or add a lightweight middleware to validate the header.

### [Q-004] `minio` Import in `gdpr.go` but Used Only for `RemoveObjectOptions{}`
- **File**: `services/ingestion/internal/handlers/gdpr.go`
- **Line**: 14
- **Issue**: The package is imported solely to pass an empty struct literal. This is acceptable but could be avoided if the handler used a helper method or the unused `storage` wrapper.

---

## Checklist Summary

| Check | Status | Notes |
|-------|--------|-------|
| Consistent error handling pattern | ❌ FAIL | `continue` vs. 500 abort is inconsistent ([ERR-002]) |
| No unused imports | ✅ PASS | All imports are consumed |
| No unused variables | ✅ PASS | No unused variables detected |
| Proper HTTP status codes | ⚠️ PARTIAL | `ListSessions` query-param bypass returns 400 instead of 403; some scan errors return no status at all |
| Request/response validation | ⚠️ PARTIAL | Gin binding used for JSON; no custom validation rules; no Content-Type enforcement |
| Logging consistency | ❌ FAIL | `log` package used only in `main.go` and `config.go`; handlers do not log errors |
| No duplicate code | ❌ FAIL | CORS, auth middleware, and chunk upload patterns duplicated across services |

**Findings Count**:  
- Lint Issues: **2**  
- Dead Code: **2**  
- Style Issues: **3**  
- Error Handling Issues: **5**  
- Test Coverage Gaps: **3**
