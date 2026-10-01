// k6 load test for the edge API (plan §7).
//
//   k6 run scripts/loadtest.js
//   BASE_URL=http://localhost:8080 VUS=50 DURATION=30s k6 run scripts/loadtest.js
//
// /db/time shows both paths depending on the API's CACHE_TTL_SECONDS:
//   - normal TTL (e.g. 30s) -> after the first request, served from Redis
//     (cache-HIT path: the fast case, CPU/network bound on the phone).
//   - CACHE_TTL_SECONDS=0    -> every request is a Postgres round-trip
//     (cache-MISS path: measures the real DB round-trip cost).

import http from 'k6/http';
import { check } from 'k6';

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
  vus: Number(__ENV.VUS || 20),
  duration: __ENV.DURATION || '20s',
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<200'],
  },
};

export default function () {
  const res = http.get(`${BASE}/db/time`);
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
