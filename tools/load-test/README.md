# Load Testing

## Quick Start

```bash
# Run locally (requires k6 installed)
k6 run load-test.js

# Run with Docker
docker-compose -f docker-compose.k6.yml up

# Run smoke test
k6 run smoke-test.js

# Run stress test
k6 run stress-test.js
```

## Environment Variables

- `BASE_URL`: API base URL (default: http://localhost:8080/v1)
- `API_KEY`: API key for authentication (default: acad389951a6aa7659c8315a796f91e9)

## Thresholds

- p95 latency < 500ms
- Error rate < 1%
