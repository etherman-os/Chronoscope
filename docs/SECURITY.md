# Security Policy

The Chronoscope team takes security seriously. This document outlines supported versions, how to report vulnerabilities, built-in security features, and deployment best practices.

---

## Supported Versions

| Version | Status       |
|---------|--------------|
| 0.1.x   | Supported    |
| < 0.1.0 | Unsupported  |

---

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Email**: `security@chronoscope.dev` (placeholder -- update before production use)
2. **Do not** open a public issue for security bugs.
3. Include a detailed description, steps to reproduce, and potential impact.
4. Allow up to 72 hours for an initial response.

We will:
- Acknowledge receipt within 72 hours
- Provide a timeline for a fix
- Coordinate disclosure once the fix is released

---

## Security Features

### Authentication & Authorization
- **API key authentication** -- All ingestion and analytics endpoints require a valid `X-API-Key` header.
- **API key hashing** -- Keys are stored as bcrypt hashes in PostgreSQL.

### Data Protection
- **PII masking** -- Automatic detection and redaction of credit cards, emails, and passwords in captured frames.
- **GDPR compliance** -- Export and right-to-be-forgotten endpoints with audit logging.
- **Frame redaction** -- Real-time blur, blackout, or replacement of sensitive screen regions.

### Infrastructure Hardening
- **Rate limiting** -- Configurable per-endpoint rate limits to prevent abuse.
- **CORS restrictions** -- Strict origin allowlists for browser-based clients.
- **Input validation** -- Strict schema validation on all ingestion payloads.
- **Security audit** -- v0.1.0 underwent a full security audit; 12 CRITICAL and 26 HIGH findings were resolved.

---

## Deployment Best Practices

1. **Use TLS everywhere** -- Terminate TLS at your load balancer and use HTTPS for all SDK-to-API communication.
2. **Rotate API keys** -- Rotate project API keys every 90 days.
3. **Restrict network access** -- Place PostgreSQL, Redis, and MinIO on a private network; expose only the load balancer and ingestion API.
4. **Enable audit logging** -- Keep `audit_logs` enabled and ship logs to a SIEM.
5. **Run least-privilege** -- Run containers as non-root users.
6. **Keep dependencies updated** -- Monitor Go, Rust, and Node.js dependencies for CVEs.
7. **Backup encrypted** -- Encrypt PostgreSQL and MinIO backups at rest.
