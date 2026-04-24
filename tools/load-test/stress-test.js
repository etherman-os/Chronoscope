import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/v1';
const API_KEY = __ENV.API_KEY || 'dev-api-key-12345';

export const options = {
  stages: [
    { duration: '2m', target: 200 },    // Ramp up to moderate load
    { duration: '3m', target: 500 },    // Continue ramping
    { duration: '5m', target: 2000 },   // Peak stress load
    { duration: '5m', target: 2000 },   // Sustain peak
    { duration: '2m', target: 0 },      // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],   // 95th percentile < 500ms
    http_req_failed: ['rate<0.01'],     // Error rate < 1%
  },
};

export default function () {
  const headers = {
    'X-API-Key': API_KEY,
    'Content-Type': 'application/json',
  };

  // 1. Session Init
  const initRes = http.post(`${BASE_URL}/sessions/init`, JSON.stringify({
    user_id: `user_${__VU}`,
    capture_mode: 'hybrid',
    metadata: { os_version: 'test', app_version: '1.0.0' },
  }), { headers });

  check(initRes, {
    'init status is 201': (r) => r.status === 201,
  });

  const sessionId = initRes.json('session_id');
  if (!sessionId) return;

  sleep(1);

  // 2. Chunk Upload (simulated)
  const chunkData = open('../fixtures/test-chunk.jpg', 'b');
  const chunkRes = http.post(`${BASE_URL}/sessions/${sessionId}/chunks`, {
    chunk: http.file(chunkData || new Uint8Array(1024), 'chunk.jpg', 'image/jpeg'),
  }, {
    headers: {
      'X-API-Key': API_KEY,
      'X-Chunk-Index': '0',
    },
  });

  check(chunkRes, {
    'chunk upload status is 200': (r) => r.status === 200,
  });

  sleep(0.5);

  // 3. Event Batch Upload
  const eventsRes = http.post(`${BASE_URL}/sessions/${sessionId}/events`, JSON.stringify({
    events: [
      { event_type: 'click', timestamp_ms: 1000, x: 100, y: 200, target: 'button#submit', payload: '{}' },
      { event_type: 'scroll', timestamp_ms: 2000, x: 0, y: 500, target: '', payload: '{}' },
      { event_type: 'input', timestamp_ms: 3000, x: 50, y: 50, target: 'input#email', payload: '{"value":"test@example.com"}' },
    ],
  }), { headers });

  check(eventsRes, {
    'events upload status is 200': (r) => r.status === 200,
  });

  sleep(0.5);

  // 4. Session Complete
  const completeRes = http.post(`${BASE_URL}/sessions/${sessionId}/complete`, '{}', { headers });

  check(completeRes, {
    'complete status is 200': (r) => r.status === 200,
  });

  sleep(1);
}
