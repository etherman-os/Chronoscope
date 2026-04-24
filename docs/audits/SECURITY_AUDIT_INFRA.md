# Infrastructure Security Audit Report

**Project**: Chronoscope  
**Auditor**: Security & Quality Auditor (Infrastructure)  
**Scope**: `services/web/`, `services/landing/`, `docker/`, `.github/workflows/`, `Makefile`, Dockerfiles  
**Date**: 2026-04-25

---

## Executive Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 1 |
| HIGH     | 10 |
| MEDIUM   | 10 |
| LOW      | 10 |

---

## CRITICAL Findings

### [C-001] API Key Exposed in Client Bundle with Weak Fallback
- **File**: `services/web/src/api/client.ts`
- **Line**: 5
- **Code**:
  ```typescript
  const API_KEY = import.meta.env.VITE_API_KEY || 'dev-api-key-12345';
  ```
- **Issue**: The dashboard embeds an API key directly into the compiled client bundle. Because `VITE_` prefixed env vars are inlined at build time, the key is visible in plaintext in the generated JavaScript. The fallback value `dev-api-key-12345` is a hardcoded, predictable secret.
- **Impact**: Any end user can open DevTools → Network or Sources, extract the `X-API-Key` header value, and replay API requests to list all sessions or fetch arbitrary session details. This constitutes a direct authentication bypass for read-only data exfiltration.
- **Fix**: Remove client-side API key authentication entirely. Proxy all API calls through a lightweight backend (e.g., a Next.js API route or a thin reverse-proxy) that stores the key server-side. Alternatively, switch to short-lived, httpOnly cookie-based sessions.

---

## HIGH Findings

### [H-001] No `.dockerignore` Files — Build Context Leaks Secrets
- **File**: `services/analytics/Dockerfile`, `services/ingestion/Dockerfile`, `services/processor/Dockerfile`
- **Line**: `COPY . .` in each Dockerfile
- **Issue**: None of the service directories contain a `.dockerignore`. The `COPY . .` instruction copies the entire build context, including potential `.env` files, `.git` directories, local secrets, IDE configs, and `node_modules`.
- **Impact**: Sensitive files can be permanently baked into Docker image layers. Even if removed in a later layer, they remain in the layer history and can be extracted by anyone with image access.
- **Fix**: Add a `.dockerignore` to each service directory:
  ```
  .env
  .env.*
  .git
  .gitignore
  *.md
  .vscode/
  .idea/
  node_modules/
  target/
  dist/
  ```

### [H-002] CI Pipeline Missing Frontend Build & Test Jobs
- **File**: `.github/workflows/ci.yml`
- **Line**: N/A (missing jobs)
- **Issue**: The CI workflow runs tests for `services/ingestion`, `packages/sdk-macos`, and checks script executability, but there are **zero** jobs for `services/web` or `services/landing`.
- **Impact**: Frontend build breakages, TypeScript errors, dependency incompatibilities, or security vulnerabilities in npm packages are never caught before merge.
- **Fix**: Add dedicated jobs:
  ```yaml
  web:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: cd services/web && npm ci && npm run build && npm run lint
  landing:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - run: cd services/landing && npm ci && npm run build && npm run lint
  ```

### [H-003] CI Lint Job Does Not Actually Lint Code
- **File**: `.github/workflows/ci.yml`
- **Line**: 25-28
- **Code**:
  ```yaml
  - name: Check scripts are executable
    run: |
      test -x scripts/bump-version.sh
      test -x scripts/release.sh
  ```
- **Issue**: The job named "Lint" only verifies shell script permissions. It does **not** run `golangci-lint`, `eslint`, `tsc`, or any other code linter.
- **Impact**: Linting rules are effectively ignored in CI, allowing style violations, unused variables, and potential bugs to reach `main`.
- **Fix**: Rename the job to "Check Scripts" and add a real lint job that runs `golangci-lint run`, `npm run lint` (after fixing H-004), and `cargo clippy`.

### [H-004] Missing ESLint Dependency Breaks `npm run lint`
- **File**: `services/web/package.json`
- **Line**: 10, 18-24
- **Code**:
  ```json
  "lint": "eslint . --ext ts,tsx"
  ```
- **Issue**: The `lint` script invokes `eslint`, but `eslint` (and `@typescript-eslint/*` plugins) is absent from `devDependencies`.
- **Impact**: On a fresh clone, `npm install && npm run lint` fails with "command not found". Developers will skip linting.
- **Fix**: Add to `devDependencies`:
  ```json
  "eslint": "^8.57.0",
  "@typescript-eslint/eslint-plugin": "^7.0.0",
  "@typescript-eslint/parser": "^7.0.0"
  ```

### [H-005] Floating `latest` Tag Used for Base Images
- **File**: `docker/docker-compose.yml` (line 32), `services/analytics/Dockerfile` (line 13), `services/ingestion/Dockerfile` (line 13)
- **Code**:
  ```yaml
  image: minio/minio:latest
  ```
  ```dockerfile
  FROM alpine:latest
  ```
- **Issue**: The `latest` tag is mutable. A future pull may retrieve a different image with breaking changes or newly introduced vulnerabilities.
- **Impact**: Non-reproducible builds and increased supply-chain attack surface.
- **Fix**: Pin to immutable references:
  ```yaml
  image: minio/minio:RELEASE.2024-04-25T
  ```
  ```dockerfile
  FROM alpine:3.19
  ```

### [H-006] GitHub Workflow Permissions Are Implicit (Default Write-All)
- **File**: `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- **Issue**: Neither workflow defines a `permissions` block. For private repositories, the default `GITHUB_TOKEN` scope is effectively `write-all`.
- **Impact**: A compromised third-party action or dependency could modify repository contents, create releases, or push tags without explicit authorization.
- **Fix**: Apply the principle of least privilege.
  ```yaml
  # ci.yml
  permissions:
    contents: read
  ```
  ```yaml
  # release.yml
  permissions:
    contents: write
  ```

### [H-007] Deprecated `actions/create-release` Action
- **File**: `.github/workflows/release.yml`
- **Line**: 24
- **Code**:
  ```yaml
  - uses: actions/create-release@v1
  ```
- **Issue**: GitHub has archived `actions/create-release`. It receives no security patches and may break without notice.
- **Impact**: Unmaintained dependency in the release pipeline.
- **Fix**: Replace with `softprops/action-gh-release@v2` or use the native `gh release create` CLI.

### [H-008] No Content Security Policy (CSP) Configured
- **File**: `services/web/index.html`, `services/landing/src/app/layout.tsx`
- **Issue**: Neither the Vite dashboard nor the Next.js landing page sets a Content-Security-Policy header or `<meta>` tag.
- **Impact**: CSP is a critical defense-in-depth layer against XSS. Without it, a future XSS vulnerability (e.g., via a compromised npm dependency) would be trivially exploitable.
- **Fix**:
  - **Web**: Add to `index.html` `<head>`:
    ```html
    <meta http-equiv="Content-Security-Policy"
          content="default-src 'self'; connect-src 'self' http://localhost:8080; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:;">
    ```
  - **Landing**: Add headers in `next.config.js`:
    ```js
    async headers() {
      return [{
        source: '/:path*',
        headers: [{
          key: 'Content-Security-Policy',
          value: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline';"
        }]
      }];
    }
    ```

### [H-009] Missing `package-lock.json` for Landing Page
- **File**: `services/landing/package.json`
- **Issue**: No `package-lock.json` is present in the landing directory, yet `node_modules` is gitignored.
- **Impact**: Builds are not reproducible. Different developers and CI runners may resolve different dependency versions, introducing unexpected bugs or vulnerable packages.
- **Fix**: Run `npm install` inside `services/landing` and commit the generated `package-lock.json`.

### [H-010] Hardcoded Project ID in SessionList
- **File**: `services/web/src/components/SessionList.tsx`
- **Line**: 9
- **Code**:
  ```typescript
  const PROJECT_ID = '22222222-2222-2222-2222-222222222222';
  ```
- **Issue**: The project identifier is baked into source code and cannot be overridden per environment.
- **Impact**: The dashboard is permanently locked to a single project. Any multi-tenant or multi-environment deployment requires a code change.
- **Fix**: Source from environment variables:
  ```typescript
  const PROJECT_ID = import.meta.env.VITE_PROJECT_ID || '';
  if (!PROJECT_ID) throw new Error('VITE_PROJECT_ID is required');
  ```

---

## MEDIUM Findings

### [M-001] Internal Services Bound to Host Ports in Compose File
- **File**: `docker/docker-compose.yml`
- **Line**: 6, 24, 39, 40
- **Issue**: PostgreSQL (`5432`), Redis (`6379`), and MinIO console (`9001`) are mapped to the host machine.
- **Impact**: In shared or production-like environments, this exposes internal infrastructure directly. The file is currently used for local dev, but without a production-specific override, developers may mistakenly use it in staging.
- **Fix**: Document that this file is dev-only, or create `docker-compose.prod.yml` that removes host port bindings for internal services and places them on an internal Docker network only.

### [M-002] No Request Timeout in Axios Client
- **File**: `services/web/src/api/client.ts`
- **Line**: 7
- **Issue**: `axios.create()` is instantiated without a `timeout` property.
- **Impact**: A hanging API request will block the UI indefinitely.
- **Fix**: Add `timeout: 10000` (10 seconds) and implement a request interceptor for cancellation tokens on unmount.

### [M-003] No Centralized Error Handling in API Client
- **File**: `services/web/src/api/client.ts`
- **Issue**: No request/response interceptors are configured. HTTP 401, 403, or 500 errors are passed raw to callers.
- **Impact**: Inconsistent UX and no global handling for auth expiration.
- **Fix**: Add interceptors to log errors, refresh tokens (if applicable), and surface user-friendly messages.

### [M-004] `setInterval` in VideoPlayer Causes Unnecessary Re-renders
- **File**: `services/web/src/components/VideoPlayer.tsx`
- **Line**: 18-28
- **Code**:
  ```typescript
  const interval = setInterval(() => {
    if (videoRef.current) {
      const timeMs = videoRef.current.currentTime * 1000;
      setCurrentTime(timeMs);
      onTimeUpdate?.(timeMs);
    }
  }, 100);
  ```
- **Issue**: The interval fires every 100ms regardless of whether the video is playing, paused, or has ended. This causes constant React re-renders.
- **Impact**: Wasted CPU/battery and potential jank on lower-end devices.
- **Fix**: Use the native `<video onTimeUpdate={...}>` event (throttled if needed) or `requestAnimationFrame` gated by `!videoRef.current.paused`.

### [M-005] Potential Workflow Command Injection via Commit Messages
- **File**: `.github/workflows/release.yml`
- **Line**: 18-21
- **Code**:
  ```yaml
  echo "CHANGELOG<<EOF" >> $GITHUB_OUTPUT
  git log --pretty=format:"- %s (%h)" $(git describe --tags --abbrev=0 HEAD~1)..HEAD >> $GITHUB_OUTPUT
  ```
- **Issue**: Commit messages (`%s`) are written directly to `$GITHUB_OUTPUT`. A malicious commit message containing newline characters or workflow command syntax could inject arbitrary workflow commands.
- **Impact**: Tampering with workflow outputs, potentially altering release artifacts.
- **Fix**: Use a dedicated changelog generation action (e.g., `mikepenz/release-changelog-builder-action`) or sanitize/encode the output before writing.

### [M-006] No `.env.example` for Web Dashboard
- **File**: `services/web/`
- **Issue**: There is no `.env.example` documenting required environment variables (`VITE_API_URL`, `VITE_API_KEY`, `VITE_PROJECT_ID`).
- **Impact**: Onboarding friction and risk of misconfiguration.
- **Fix**: Create `services/web/.env.example`:
  ```
  VITE_API_URL=http://localhost:8080/v1
  VITE_API_KEY=your-server-side-proxy-key
  VITE_PROJECT_ID=22222222-2222-2222-2222-222222222222
  ```

### [M-007] Bind Mount Uses Parent Directory Traversal
- **File**: `docker/docker-compose.yml`
- **Line**: 13
- **Code**:
  ```yaml
  volumes:
    - ../migrations:/docker-entrypoint-initdb.d:ro
  ```
- **Issue**: The relative `../migrations` path assumes the working directory is always `docker/`. Running `docker compose` from the repo root breaks the mount.
- **Impact**: Fragile local development setup.
- **Fix**: Use an explicit build context or copy migrations into a custom Postgres image with a `Dockerfile`.

### [M-008] Missing `next-env.d.ts` in Landing Page
- **File**: `services/landing/`
- **Issue**: `next-env.d.ts` is absent. Next.js relies on this file for TypeScript ambient declarations.
- **Impact**: Type errors or missing Next.js type definitions during builds.
- **Fix**: Run `npx next` to generate the file, then commit it (Next.js docs recommend committing this file).

### [M-009] No Branch Protection or Security Policy Documented
- **File**: N/A
- **Issue**: Neither `CONTRIBUTING.md` nor `docs/` describes required PR reviews, status checks, signed commits, or a `SECURITY.md` policy.
- **Impact**: Ad-hoc governance; unreviewed or malicious code can reach `main`.
- **Fix**: Document branch protection rules (require 1+ reviewer, require status checks, dismiss stale reviews) and add a `SECURITY.md` with disclosure instructions.

### [M-010] Release Script Pushes Without Verifying Remote State
- **File**: `scripts/release.sh`
- **Line**: 32-33
- **Code**:
  ```bash
  git push origin main
  git push origin "v$VERSION"
  ```
- **Issue**: The script pushes `main` unconditionally. If the local branch is behind remote, the push will be rejected. If a force-push is accidentally configured, it could overwrite history.
- **Impact**: Failed or dangerous releases.
- **Fix**: Add `git pull origin main --rebase` before push, or transition to a PR-based release flow using `gh pr create` and tag-on-merge.

---

## LOW Findings

### [L-001] Excessive Inline Styles Reduce Maintainability
- **File**: `services/web/src/pages/Dashboard.tsx`, `services/web/src/components/SessionList.tsx`, etc.
- **Issue**: Nearly all styling is done via inline `style={{...}}` objects rather than CSS Modules, Tailwind, or a design system.
- **Impact**: Harder to maintain, no dead-code elimination for CSS, and duplicated style objects on every render.
- **Fix**: Migrate to CSS Modules (`*.module.css`) or a utility-first framework like Tailwind (already used in landing).

### [L-002] `key` Prop Relies on Array Index
- **File**: `services/web/src/components/EventTimeline.tsx` (line 38), `services/web/src/components/VideoPlayer.tsx` (line 69)
- **Code**:
  ```tsx
  key={`${event.timestamp_ms}-${index}`}
  ```
- **Issue**: Using `index` as part of the React key reduces the effectiveness of reconciliation if the array is reordered or filtered.
- **Impact**: Potential stale UI state or unnecessary DOM mutations.
- **Fix**: Use a stable unique property (e.g., `event.id` if it exists). If not available, generate a UUID when normalizing API data.

### [L-003] Interactive Divs Lack Accessibility Attributes
- **File**: `services/web/src/components/SessionList.tsx`
- **Line**: 85-100
- **Issue**: Clickable session rows are `<div>` elements without `role="button"`, `tabIndex`, or `aria-label`.
- **Impact**: Screen readers and keyboard-only users cannot navigate the session list.
- **Fix**: Convert to `<button>` or add:
  ```tsx
  role="button"
  tabIndex={0}
  aria-label={`Session for ${session.user_id || 'Anonymous'}`}
  onKeyDown={(e) => e.key === 'Enter' && onSelect(session)}
  ```

### [L-004] `sed -i` Portability Issue in Shell Scripts
- **File**: `scripts/bump-version.sh`
- **Line**: 15, 17, 21, 24, 27
- **Issue**: `sed -i` without a backup extension fails on macOS BSD `sed`.
- **Impact**: Release scripts are not portable across macOS and Linux dev environments.
- **Fix**: Use `sed -i.bak` followed by `rm -f *.bak`, or switch to `perl -pi -e`.

### [L-005] No `HEALTHCHECK` in Dockerfiles
- **File**: All Dockerfiles (`services/analytics/Dockerfile`, `services/ingestion/Dockerfile`, `services/processor/Dockerfile`)
- **Issue**: None of the Dockerfiles define a `HEALTHCHECK` instruction.
- **Impact**: Orchestrators (Docker Compose, Kubernetes) cannot automatically restart unhealthy containers.
- **Fix**: Add to each Dockerfile:
  ```dockerfile
  HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/healthz || exit 1
  ```

### [L-006] Makefile `proto` Target Lacks Prerequisite Validation
- **File**: `Makefile`
- **Line**: 16-25
- **Issue**: The `proto` target invokes `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, and `protoc-gen-swift` without checking their presence.
- **Impact**: Cryptic error messages if a contributor hasn't installed the protobuf toolchain.
- **Fix**: Add guard clauses:
  ```makefile
  proto:
  	@command -v protoc >/dev/null 2>&1 || { echo "protoc is required"; exit 1; }
  	...
  ```

### [L-007] Makefile `test` Target Stops on First Failure
- **File**: `Makefile`
- **Line**: 28-34
- **Issue**: Because Make executes each line in a fresh shell and stops on the first non-zero exit, if ingestion tests fail, analytics and SDK tests never run.
- **Impact**: Incomplete feedback for developers fixing cross-service issues.
- **Fix**: Use a script that aggregates exit codes, or append `|| TEST_FAILED=1` to each line and exit with `exit $$TEST_FAILED`.

### [L-008] Button Without Explicit `type` Attribute
- **File**: `services/landing/src/components/PricingCard.tsx`
- **Line**: 41
- **Code**:
  ```tsx
  <button className="...">
  ```
- **Issue**: No `type="button"` is specified. If this component is ever nested inside a `<form>`, it will submit the form.
- **Impact**: Unexpected form submission behavior.
- **Fix**: Add `type="button"`.

### [L-009] Error Objects Swallowed in Dashboard
- **File**: `services/web/src/pages/Dashboard.tsx`
- **Line**: 21-22
- **Code**:
  ```tsx
  } catch (err) {
    setError('Failed to load session details');
  }
  ```
- **Issue**: The actual error (`err`) is discarded; it is not logged to the console or sent to an error-monitoring service.
- **Impact**: Production debugging is significantly harder.
- **Fix**: Log the error:
  ```tsx
  } catch (err) {
    console.error('Failed to load session details:', err);
    setError('Failed to load session details');
  }
  ```

### [L-010] VideoPlayer Interval Dependency Array Incomplete
- **File**: `services/web/src/components/VideoPlayer.tsx`
- **Line**: 28
- **Code**:
  ```tsx
  }, [onTimeUpdate]);
  ```
- **Issue**: `videoRef.current` is accessed inside the interval but is not listed in dependencies (and should not be, since refs are mutable). However, if `onTimeUpdate` changes frequently, the interval is recreated repeatedly.
- **Impact**: Slight performance overhead from clearing and recreating the interval.
- **Fix**: Wrap `onTimeUpdate` in `useCallback` in the parent (`Dashboard.tsx`) to ensure referential stability.

---

## ✅ PASS Items

### Frontend Security
- [x] **No `dangerouslySetInnerHTML` usage** — No occurrences found in `services/web/` or `services/landing/`. All DOM content is rendered via safe JSX interpolation.
- [x] **XSS prevention: all user input escaped in DOM** — User-provided strings (e.g., `session.user_id`, `event.event_type`) are rendered inside JSX `{}` expressions, which React automatically escapes.
- [x] **No sensitive data logged to console** — No `console.log` statements printing tokens, PII, or API responses were found.

### Docker Security
- [x] **Multi-stage builds used properly** — All three Dockerfiles (`analytics`, `ingestion`, `processor`) correctly separate build-time dependencies from runtime images, minimizing final image size and attack surface.
- [x] **No bind mounts to sensitive host paths** — The only bind mount (`../migrations:/docker-entrypoint-initdb.d:ro`) points to a read-only application directory, not `/etc`, `/root`, or Docker socket paths.

### CI/CD Security
- [x] **No hardcoded secrets in workflow files** — `secrets.GITHUB_TOKEN` is the only secret referenced, and no API keys or passwords are embedded in YAML.
- [x] **Artifact retention configured** — No artifacts are uploaded in current workflows, so retention is not applicable.

### Quality
- [x] **Consistent code style (TypeScript strict mode)** — Both `services/web/tsconfig.json` and `services/landing/tsconfig.json` enable `"strict": true`, ensuring strong typing.
- [x] **Build reproducibility (web)** — `services/web/package-lock.json` exists, locking dependency versions.

---

## Recommendations (Priority Order)

1. **Immediately** proxy API calls server-side or switch to cookie auth to eliminate [C-001].
2. **This sprint** add `.dockerignore` files ([H-001]), pin image tags ([H-005]), and add explicit workflow permissions ([H-006]).
3. **Next sprint** bring frontend services into CI ([H-002], [H-003], [H-004]) and add CSP headers ([H-008]).
4. **Before next release** replace the deprecated release action ([H-007]) and commit `package-lock.json` for landing ([H-009]).
5. **Ongoing** document branch protection rules ([M-009]) and add `HEALTHCHECK` instructions to all images ([L-005]).
