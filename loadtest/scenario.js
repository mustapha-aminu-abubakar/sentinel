import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const checkLatency = new Trend('check_latency');
const checkErrors = new Rate('check_errors');

export function setup() {
  const clientRes = http.post('http://nginx:80/clients',
    JSON.stringify({ name: 'k6-load-client' }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  const clientId = clientRes.json('id');

  http.post('http://nginx:80/rules',
    JSON.stringify({
      client_id: clientId,
      api: 'k6-load',
      requests_allowed: 15000,
      window_seconds: 60,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  return { clientId, api: 'k6-load' };
}

export const options = {
  stages: [
    { duration: '30s', target: 10 },    // warm-up
    { duration: '1m',  target: 50 },    // moderate
    { duration: '1m',  target: 50 },    // steady
    { duration: '30s', target: 100 },   // peak
    { duration: '1m',  target: 100 },   // sustained peak
    { duration: '30s', target: 0 },     // cool-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'],
    http_req_failed:   ['rate<0.05'],
  },
};

export default function (data) {
  const res = http.post('http://nginx:80/v1/check',
    JSON.stringify({ client_id: data.clientId, api: data.api }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  check(res, { 'status is 200': (r) => r.status === 200 });
  checkLatency.add(res.timings.duration);
  if (res.status !== 200) checkErrors.add(1);
}
