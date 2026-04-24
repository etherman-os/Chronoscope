import http from 'k6/http';
import { check, sleep } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/v1';
const API_KEY = __ENV.API_KEY || 'dev-api-key-12345';

export const options = {
  vus: 10,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
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
