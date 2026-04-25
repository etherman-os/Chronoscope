# Security Policy

The Chronoscope team takes security seriously. This document outlines supported versions, how to report vulnerabilities, built-in security features, and deployment best practices.

---

## Supported Versions

| Version | Status       |
|---------|--------------|
| 0.2.x   | Supported    |
| 0.1.x   | End of life  |
| < 0.1.0 | Unsupported  |

---

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Email**: `security@chronoscope.dev`
2. **GitHub Private Advisory**: [Submit a private vulnerability report](https://github.com/etherman-os/chronoscope/security/advisories/new)
3. **Do not** open a public issue for security bugs.
4. Include a detailed description, steps to reproduce, and potential impact.
5. Allow up to 72 hours for an initial response.

We will:
- Acknowledge receipt within 72 hours
- Provide a timeline for a fix
- Coordinate disclosure once the fix is released

---

## Built-in Security Features

### Authentication & Authorization
- **API key authentication** — All ingestion and analytics endpoints require a valid `X-API-Key` header.
- **API key hashing** — Keys are hashed with SHA-256 before comparison against the database.
- **Project ownership checks** — Every session operation verifies the API key belongs to the project that owns the session.

### Data Protection
- **PII masking** — Automatic detection and redaction of credit cards, emails, and passwords in captured frames.
- **GDPR compliance** — Export and right-to-be-forgotten endpoints with audit logging.
- **Frame redaction** — Real-time blur, blackout, or replacement of sensitive screen regions via the Privacy Engine.

### Infrastructure Hardening
- **Rate limiting** — Configurable per-API-key rate limits (default 100 req/min).
- **CORS restrictions** — Strict origin allowlist via the `CORS_ALLOWED_ORIGIN` environment variable.
- **Input validation** — Strict validation on all ingestion payloads including max chunk size (2 MiB), max chunk index (10000), and max event batch size (1000).
- **Request size limits** — Ingestion API limits request bodies to 8 MiB; Analytics API limits to 1 MiB.
- **Security audit** — v0.1.0 underwent a full security audit; findings were resolved in v0.2.0.

---

## Deployment Hardening Checklist

1. **Use TLS everywhere** — Terminate TLS at your load balancer and use HTTPS for all SDK-to-API communication.
2. **Rotate API keys** — Rotate project API keys every 90 days.
3. **Restrict network access** — Place PostgreSQL, Redis, and MinIO on a private network; expose only the load balancer and ingestion API.
4. **Enable audit logging** — Keep `audit_logs` enabled and ship logs to a SIEM.
5. **Run least-privilege** — Run containers as non-root users (the provided Dockerfiles use `USER appuser`).
6. **Keep dependencies updated** — Monitor Go, Rust, and Node.js dependencies for CVEs.
7. **Backup encrypted** — Encrypt PostgreSQL and MinIO backups at rest.
8. **Set strong secrets** — Use cryptographically random values for `MINIO_ROOT_PASSWORD`, database passwords, and API keys.
9. **Configure CORS** — Set `CORS_ALLOWED_ORIGIN` to your exact domain; do not use wildcards in production.
