import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 10, // 10 virtual users
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests must complete below 2s
    http_req_failed: ['rate<0.1'], // Error rate must be below 10%
  },
};

const BASE_URL = __ENV.API_BASE_URL || 'https://api.finaro.finance';

export default function () {
  // Test health endpoint
  let healthResponse = http.get(`${BASE_URL}/health`);
  check(healthResponse, {
    'health status is 200': (r) => r.status === 200,
    'health response time < 500ms': (r) => r.timings.duration < 500,
  });

  sleep(1);

  // Test API v1 health if it exists
  let apiHealthResponse = http.get(`${BASE_URL}/api/v1/health`);
  check(apiHealthResponse, {
    'api health status is 200 or 404': (r) => r.status === 200 || r.status === 404,
  });

  sleep(1);
}