# Contributing to Chronoscope

Thank you for your interest in contributing! This guide will help you get started.

---

## Prerequisites

| Tool                      | Version | Purpose                          |
|---------------------------|---------|----------------------------------|
| Go                        | 1.22+   | Ingestion & Analytics APIs       |
| Rust                      | 1.75+   | Processor, Privacy Engine, Linux SDK |
| Swift                     | 5.9+    | macOS SDK                        |
| Node.js                   | 20+     | Web Dashboard & Landing Page     |
| PostgreSQL                | 16      | Database                         |
| Redis                     | 7       | Cache / Queue                    |
| MinIO                     | Latest  | Object Storage                   |
| Docker & Docker Compose   | Latest  | Local infrastructure             |
| Protobuf Compiler (`protoc`) | 3.x  | Schema generation                |

---

## Repository Structure

```
chronoscope/
├── services/
│   ├── ingestion/         # Go ingestion API
│   ├── analytics/         # Go analytics API
│   ├── processor/         # Rust video processor
│   ├── privacy-engine/    # Rust privacy engine C ABI
│   ├── web/               # React dashboard (Vite)
│   └── landing/           # Next.js landing page
├── packages/
│   ├── sdk-macos/         # Swift macOS SDK
│   ├── sdk-linux/         # Rust Linux SDK
│   └── sdk-windows/       # C++ Windows SDK
├── protocols/             # Protobuf schemas
├── migrations/            # PostgreSQL migrations
├── docker/                # Docker Compose files
└── docs/                  # Documentation
```

---

## Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/etherman-os/chronoscope.git
cd chronoscope
```

### 2. Start Local Infrastructure

```bash
make up
```

This starts PostgreSQL, Redis, and MinIO via Docker Compose.

### 3. Environment Configuration

Each Go service requires a `.env` file. Copy the examples and adjust as needed:

```bash
cp services/ingestion/.env.example services/ingestion/.env
cp services/analytics/.env.example services/analytics/.env
```

### 4. Run Services Locally

**Ingestion API** (port 8080):
```bash
cd services/ingestion
go run cmd/server/main.go
```

**Analytics API** (port 8081):
```bash
cd services/analytics
go run cmd/server/main.go
```

**Web Dashboard** (port 5173):
```bash
cd services/web
npm install
npm run dev
```

**Landing Page** (port 3000):
```bash
cd services/landing
npm install
npm run dev
```

**Video Processor**:
```bash
cd services/processor
cargo run
```

---

## Testing

Run the full test suite:

```bash
make test
```

### Per-language Commands

**Go services:**
```bash
cd services/ingestion && go test ./...
cd services/analytics && go test ./...
```

**Rust services:**
```bash
cd services/processor && cargo test
cd services/privacy-engine && cargo test
```

**macOS SDK:**
```bash
cd packages/sdk-macos && swift test
```

**Linux SDK:**
```bash
cd packages/sdk-linux && cargo test
```

**Web Dashboard:**
```bash
cd services/web && npm ci && npm run lint && npm test
```

**Landing Page:**
```bash
cd services/landing && npm ci && npm run lint && npm run build
```

---

## Code Style

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go).
- Use `gofmt` and `golangci-lint`.
- Keep functions short and focused.
- Error messages should be lowercase without punctuation.

```bash
cd services/ingestion && golangci-lint run ./...
cd services/analytics && golangci-lint run ./...
```

### Rust

- Follow the [Rust API Guidelines](https://rust-lang.github.io/api-guidelines/).
- Use `rustfmt` and `clippy`.
- Prefer `Result` over `panic!` in library code.
- Document all public APIs with `///`.

```bash
cargo fmt
cargo clippy --all-targets --all-features -- -D warnings
```

### Swift

- Follow [Swift API Design Guidelines](https://www.swift.org/documentation/api-design-guidelines/).
- Use `swift-format` if available.
- Prefer `let` over `var`.
- Use `guard` for early exits.

### TypeScript / React

- Follow the project's ESLint configuration.
- Use functional components with hooks.
- Prefer `const` and explicit types for function parameters.

```bash
cd services/web && npm run lint
cd services/landing && npm run lint
```

---

## Commit Message Conventions

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

| Type       | Description                                          |
|------------|------------------------------------------------------|
| `feat`     | New feature                                          |
| `fix`      | Bug fix                                              |
| `docs`     | Documentation only                                   |
| `style`    | Code style (formatting, missing semi-colons, etc.)   |
| `refactor` | Code refactoring                                     |
| `perf`     | Performance improvement                              |
| `test`     | Adding or correcting tests                           |
| `chore`    | Build process or auxiliary tool changes              |
| `security` | Security fix or improvement                          |

### Scopes

Common scopes: `ingestion`, `analytics`, `processor`, `privacy`, `web`, `landing`, `sdk-macos`, `sdk-windows`, `sdk-linux`, `docs`, `ci`.

### Examples

```
feat(ingestion): add chunked upload resume support

fix(sdk-macos): prevent memory leak in capture buffer

docs(readme): update quick start instructions

security(ingestion): add rate limiting per API key
```

---

## Pull Request Process

1. **Fork** the repository and create a feature branch from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```

2. **Make your changes**. Ensure they follow our style guidelines and include tests.

3. **Run the test suite** locally:
   ```bash
   make test
   make lint
   ```

4. **Update documentation** if your changes affect the public API, deployment, or architecture.

5. **Commit** using conventional commit messages.

6. **Push** your branch and open a Pull Request against `main`.

7. **Fill out the PR template** with:
   - What changed and why
   - How to test it
   - Screenshots (for UI changes)

8. **Code Review**:
   - At least one approving review is required.
   - All CI checks must pass.
   - Address review feedback promptly and respectfully.

9. **Merge**: Maintainers will squash and merge once approved.

## Branch Protection

The `main` branch is protected. All changes must go through a Pull Request with:
- At least one approving review from a maintainer.
- All required status checks passing (CI build, lint, tests).
- Stale reviews are dismissed when new commits are pushed.
- Signed commits are encouraged but not strictly required.

---

## Getting Help

- Open a [GitHub Discussion](https://github.com/etherman-os/chronoscope/discussions) for questions.
- Open an [Issue](https://github.com/etherman-os/chronoscope/issues) for bugs or feature requests.

We appreciate your contributions!
