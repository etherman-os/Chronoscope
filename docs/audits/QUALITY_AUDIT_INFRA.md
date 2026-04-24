# Infrastructure Quality Audit Report

**Project**: Chronoscope  
**Auditor**: Security & Quality Auditor (Infrastructure)  
**Scope**: `services/web/`, `services/landing/`, `docker/`, `.github/workflows/`, `Makefile`, Dockerfiles  
**Date**: 2026-04-25

---

## Executive Summary

| Category | Issues Found | Passing Checks |
|----------|--------------|----------------|
| Frontend Code Quality | 8 | 4 |
| Docker Security | 4 | 2 |
| CI/CD Pipeline Quality | 5 | 2 |
| Build Script Quality | 5 | 1 |
| Dependency Management | 2 | 1 |
| Documentation Gaps | 4 | 0 |

---

## Frontend Code Quality

### React Dashboard (`services/web/`)

#### ❌ Missing ESLint Toolchain
- **File**: `services/web/package.json`
- **Issue**: The `lint` script references `eslint`, but `eslint` and its TypeScript plugins are absent from `devDependencies`. This makes the lint script broken on fresh installs.
- **Severity**: High
- **Fix**: Add `eslint`, `@typescript-eslint/parser`, and `@typescript-eslint/eslint-plugin` to `devDependencies`. Include an `.eslintrc.cjs` extending `eslint:recommended` and `plugin:@typescript-eslint/recommended`.

#### ❌ No Unit or Integration Tests
- **File**: `services/web/`
- **Issue**: No test runner (Vitest, Jest, Cypress, Playwright) is configured. The CI job (which is also missing) therefore cannot catch regressions.
- **Severity**: High
- **Fix**: Add Vitest (aligned with Vite) and React Testing Library. Write at minimum smoke tests for `Dashboard.tsx` and `SessionList.tsx`.

#### ❌ Excessive Inline Styles
- **Files**: `services/web/src/pages/Dashboard.tsx`, `services/web/src/components/SessionList.tsx`, `services/web/src/components/EventTimeline.tsx`, `services/web/src/components/VideoPlayer.tsx`
- **Issue**: Dozens of inline `style={{...}}` declarations clutter components, make them harder to read, and prevent style reuse. There is no CSS Modules, styled-components, or Tailwind usage in the dashboard.
- **Severity**: Medium
- **Fix**: Adopt Tailwind (already used in landing) or migrate to CSS Modules for scoped, reusable styles.

#### ❌ Missing Error Boundaries
- **File**: `services/web/src/main.tsx`
- **Issue**: The root render does not wrap `<App />` in an Error Boundary. Any unhandled exception in a child component will crash the entire dashboard (white screen).
- **Severity**: Medium
- **Fix**: Add a top-level `<ErrorBoundary>` (e.g., `react-error-boundary`) with a fallback UI.

#### ❌ Swallowed Errors in Async Handlers
- **File**: `services/web/src/pages/Dashboard.tsx` (line 21-22), `services/web/src/components/SessionList.tsx` (line 22-23)
- **Issue**:
  ```tsx
  } catch (err) {
    setError('Failed to load ...');
  }
  ```
  The caught error is never logged or reported.
- **Severity**: Low
- **Fix**: Add `console.error(err)` or integrate an error reporting SDK (Sentry, etc.).

#### ❌ VideoPlayer Uses `setInterval` Instead of Native Events
- **File**: `services/web/src/components/VideoPlayer.tsx`
- **Issue**: A 100ms `setInterval` polls `videoRef.current.currentTime` continuously, even when the video is paused. This is inefficient and can cause frame drops.
- **Severity**: Medium
- **Fix**: Replace with the native `<video onTimeUpdate={...}>` event.

#### ❌ `key` Props Use Array Index
- **File**: `services/web/src/components/EventTimeline.tsx`, `services/web/src/components/VideoPlayer.tsx`
- **Issue**: React keys are constructed as `` `${event.timestamp_ms}-${index}` ``. If events are filtered or reordered, React may reuse DOM nodes incorrectly.
- **Severity**: Low
- **Fix**: Use a stable unique ID. If the API does not provide one, generate a UUID during data normalization.

#### ❌ Accessibility (a11y) Deficiencies
- **File**: `services/web/src/components/SessionList.tsx`
- **Issue**: Clickable session rows are `<div>` elements without `role`, `tabIndex`, or keyboard handlers.
- **Severity**: Low
- **Fix**: Convert to `<button>` or add proper ARIA attributes and `onKeyDown` handlers.

### Next.js Landing Page (`services/landing/`)

#### ❌ No `next-env.d.ts` Committed
- **File**: `services/landing/`
- **Issue**: Next.js auto-generates `next-env.d.ts` for ambient type declarations. Its absence can cause TypeScript compilation errors in CI or for new contributors.
- **Severity**: Medium
- **Fix**: Generate and commit the file.

#### ✅ PASS — Modern Tooling
- The landing page uses Next.js 14 App Router, TypeScript strict mode, and Tailwind CSS. Code is well-organized into components (`Navbar`, `FeatureCard`, `PricingCard`, `WaitlistForm`).

#### ✅ PASS — No Unused Dependencies
- `package.json` dependencies are minimal: `next`, `react`, `react-dom`, plus standard Tailwind toolchain.

---

## Docker Security

#### ❌ No `.dockerignore` Files
- **File**: `services/analytics/Dockerfile`, `services/ingestion/Dockerfile`, `services/processor/Dockerfile`
- **Issue**: `COPY . .` copies the entire build context. Local `.env` files, `.git` history, IDE configs, and OS artifacts can be baked into image layers.
- **Severity**: High
- **Fix**: Add `.dockerignore` to each service:
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

#### ❌ Containers Run as Root
- **File**: All Dockerfiles
- **Issue**: No `USER` instruction is present. All processes execute as `root` (UID 0).
- **Severity**: High
- **Fix**: Create a non-root user in the final stage:
  ```dockerfile
  RUN adduser -D -u 1000 appuser
  USER appuser
  ```

#### ❌ Floating `latest` Image Tags
- **File**: `docker/docker-compose.yml` (line 32), `services/analytics/Dockerfile` (line 13), `services/ingestion/Dockerfile` (line 13)
- **Issue**: `minio/minio:latest` and `alpine:latest` are mutable tags. Future builds may silently ingest breaking changes or new vulnerabilities.
- **Severity**: High
- **Fix**: Pin to immutable versions or digests.

#### ❌ No `HEALTHCHECK` Instructions
- **File**: All Dockerfiles
- **Issue**: None define a `HEALTHCHECK`. Orchestrators cannot detect stuck or deadlocked containers.
- **Severity**: Low
- **Fix**: Add a `HEALTHCHECK CMD` that probes `/healthz` or similar.

#### ✅ PASS — Multi-Stage Builds
- All Dockerfiles correctly separate compilation from runtime, reducing final image size and limiting the runtime attack surface.

#### ✅ PASS — Minimal Runtime Dependencies
- The Go images only add `ca-certificates`. The Rust image only adds `ffmpeg` (required for the processor).

---

## CI/CD Pipeline Quality

#### ❌ No Frontend Jobs in CI
- **File**: `.github/workflows/ci.yml`
- **Issue**: The workflow tests Go ingestion and Swift macOS SDK, but completely omits `services/web` and `services/landing`. Build failures, type errors, or dependency vulnerabilities in the frontend go undetected.
- **Severity**: High
- **Fix**: Add `web` and `landing` jobs that run `npm ci`, `npm run build`, and `npm run lint`.

#### ❌ Lint Job Does Not Lint Code
- **File**: `.github/workflows/ci.yml` (lines 25-28)
- **Issue**: The job named "Lint" only verifies shell script permissions (`test -x`). It never invokes `eslint`, `tsc --noEmit`, `golangci-lint`, or `cargo clippy`.
- **Severity**: High
- **Fix**: Rename the job to "Script Permissions" and add a real lint matrix job.

#### ❌ Deprecated Release Action
- **File**: `.github/workflows/release.yml` (line 24)
- **Issue**: `actions/create-release@v1` is archived by GitHub and receives no updates.
- **Severity**: High
- **Fix**: Migrate to `softprops/action-gh-release@v2` or the `gh` CLI.

#### ❌ Missing Workflow Permissions
- **File**: `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- **Issue**: No `permissions` block means the default `GITHUB_TOKEN` scope is used, which is overly broad.
- **Severity**: High
- **Fix**: Explicitly declare `permissions: contents: read` in CI and `permissions: contents: write` in the release job.

#### ❌ Commit Message Injection Risk in Release Workflow
- **File**: `.github/workflows/release.yml` (lines 18-21)
- **Issue**: `git log --pretty=format:"- %s (%h)"` writes unsanitized commit messages into `$GITHUB_OUTPUT`.
- **Severity**: Medium
- **Fix**: Use a dedicated changelog action or sanitize the string.

#### ✅ PASS — No Hardcoded Secrets in Workflows
- Workflows only reference `secrets.GITHUB_TOKEN`; no API keys, passwords, or tokens are embedded.

#### ✅ PASS — Standard Action Versions
- `actions/checkout@v4`, `actions/setup-go@v5`, and `actions/setup-node@v4` are current major versions.

---

## Build Script Quality

#### ❌ `Makefile` `test` Target Stops on First Failure
- **File**: `Makefile` (lines 28-34)
- **Issue**: If ingestion tests fail, analytics and SDK tests are skipped because Make exits on the first non-zero return.
- **Severity**: Low
- **Fix**: Aggregate results with a script or use `||` guards:
  ```makefile
  test:
  	@failed=0; \
  	cd services/ingestion && go test ./... || failed=1; \
  	cd services/analytics && go test ./... || failed=1; \
  	cd packages/sdk-macos && swift test || failed=1; \
  	exit $$failed
  ```

#### ❌ `Makefile` `proto` Target Lacks Prerequisite Checks
- **File**: `Makefile` (lines 16-25)
- **Issue**: The target assumes `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`, and `protoc-gen-swift` are installed, but does not validate them.
- **Severity**: Low
- **Fix**: Add guard clauses with clear error messages.

#### ❌ `Makefile` `up` Does Not Rebuild Changed Images
- **File**: `Makefile` (line 5)
- **Issue**: `docker compose -f docker/docker-compose.yml up -d` will use cached images even if Dockerfiles or source code have changed.
- **Severity**: Low
- **Fix**: Add `--build` flag or create a separate `up-build` target:
  ```makefile
  up:
  	docker compose -f docker/docker-compose.yml up -d --build
  ```

#### ❌ `scripts/bump-version.sh` Portability Issue
- **File**: `scripts/bump-version.sh` (lines 15, 17, 21, 24, 27)
- **Issue**: `sed -i` without a backup extension is incompatible with macOS BSD `sed`.
- **Severity**: Low
- **Fix**: Use `sed -i.bak` and clean up, or use `perl -pi -e`.

#### ❌ `scripts/release.sh` Pushes Unconditionally
- **File**: `scripts/release.sh` (lines 32-33)
- **Issue**: `git push origin main` is executed without verifying that the local branch is up-to-date with the remote.
- **Severity**: Medium
- **Fix**: Add `git pull origin main --rebase` before pushing, or adopt a tag-on-merge workflow via GitHub UI/API.

#### ✅ PASS — Makefile Uses `.PHONY` Correctly
- All non-file targets (`up`, `down`, `proto`, `test`, `lint`) are declared `.PHONY`, preventing conflicts with files of the same name.

---

## Dependency Management

#### ❌ Missing `package-lock.json` for Landing Page
- **File**: `services/landing/`
- **Issue**: No lockfile is committed. Builds are non-reproducible and vulnerable to transitive dependency drift.
- **Severity**: High
- **Fix**: Run `npm install` in `services/landing` and commit the lockfile.

#### ❌ No Dependency Audit in CI
- **File**: `.github/workflows/ci.yml`
- **Issue**: No `npm audit` or `pnpm audit` step runs for frontend services.
- **Severity**: Medium
- **Fix**: Add `npm audit --audit-level=high` to the frontend CI jobs, or integrate Dependabot.

#### ✅ PASS — Web Dashboard Has Lockfile
- `services/web/package-lock.json` exists, ensuring reproducible installs.

---

## Documentation Gaps

#### ❌ No `.env.example` for Web Dashboard
- **File**: `services/web/`
- **Issue**: Developers have no reference for required environment variables (`VITE_API_URL`, `VITE_API_KEY`, `VITE_PROJECT_ID`).
- **Severity**: Medium
- **Fix**: Create `services/web/.env.example` with dummy values and comments.

#### ❌ No Branch Protection Documentation
- **File**: `docs/CONTRIBUTING.md`
- **Issue**: There is no mention of required PR reviews, mandatory status checks, or signed commits.
- **Severity**: Medium
- **Fix**: Add a section describing the branch protection policy and how to request reviews.

#### ❌ No `SECURITY.md`
- **File**: N/A
- **Issue**: The repository lacks a security policy file, meaning researchers or users do not know how to report vulnerabilities responsibly.
- **Severity**: Medium
- **Fix**: Add a `SECURITY.md` at the repo root with contact email and disclosure timeline.

#### ❌ No Frontend Testing Documentation
- **File**: `docs/CONTRIBUTING.md`, `services/web/README.md`
- **Issue**: There are no instructions on how to run frontend tests (because there are no tests), nor how to set up the local dev environment for the dashboard beyond `npm install`.
- **Severity**: Medium
- **Fix**: Document the env vars, proxy behavior (`vite.config.ts`), and how to run the dashboard against a local backend.

---

## Recommendations

1. **Standardize linting and formatting**: Add ESLint + Prettier to both frontend services and enforce them in CI.
2. **Expand CI coverage**: Immediately add build, lint, and audit jobs for `services/web` and `services/landing`.
3. **Harden Docker artifacts**: Add `.dockerignore`, non-root `USER`, pinned base images, and `HEALTHCHECK` to all Dockerfiles.
4. **Lock dependencies**: Commit `package-lock.json` for the landing page and enable Dependabot alerts across the monorepo.
5. **Improve error resilience**: Add Error Boundaries to the React dashboard and centralize API error handling/logging.
6. **Document operational policies**: Add `SECURITY.md`, branch protection rules, and `.env.example` files for every service.
