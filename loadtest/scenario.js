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
      requests_allowed: 100000,
      window_seconds: 60,
    }),
    { headers: { 'Content-Type': 'application/json' } }
  );

  return { clientId, api: 'k6-load' };
}

export const options = {
  stages: [
    { duration: '1m',  target: 50 },    // warm-up: 50 concurrent VUs
    { duration: '2m',  target: 200 },   // ramp to moderate load
    { duration: '3m',  target: 200 },   // steady moderate
    { duration: '1m',  target: 1000 },  // ramp to high load
    { duration: '2m',  target: 1000 },  // sustained high load
    { duration: '30s', target: 2000 },  // spike
    { duration: '1m',  target: 2000 },  // sustained spike
    { duration: '30s', target: 0 },     // cool-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed:   ['rate<0.01'],
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
