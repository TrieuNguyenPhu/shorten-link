import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

const mode = __ENV.MODE || 'smoke';
if (!['smoke', 'stress'].includes(mode)) throw new Error('MODE must be smoke or stress');
if (mode === 'stress' && __ENV.ALLOW_STRESS !== '1') throw new Error('Stress requires explicit authorization and ALLOW_STRESS=1');
const base = (__ENV.BASE_URL || '').replace(/\/$/, '');
const code = __ENV.CODE || '';
if (!/^(https:\/\/[A-Za-z0-9.-]+(?::[0-9]+)?|http:\/\/(localhost|127\.0\.0\.1):[0-9]+)$/.test(base)) throw new Error('BASE_URL must be an HTTPS origin or explicit local HTTP origin');
if (!/^[A-Za-z0-9]{1,32}$/.test(code)) throw new Error('CODE must be an existing short code');

export const options = {
  maxRedirects: 0,
  thresholds: { checks: ['rate==1'], redirect_failed: ['rate==0'] },
  scenarios: mode === 'smoke'
    ? { smoke: { executor: 'constant-vus', vus: 2, duration: '10s', gracefulStop: '5s' } }
    : { stress: { executor: 'ramping-vus', startVUs: 1, stages: [
      { duration: '20s', target: 10 }, { duration: '30s', target: 40 },
      { duration: '30s', target: 40 }, { duration: '10s', target: 0 },
    ], gracefulRampDown: '5s' } },
};
const successful = new Counter('redirect_successful');
const failed = new Rate('redirect_failed');
const failures = new Counter('redirect_failures');
const throttled = new Counter('redirect_throttled');
const serverErrors = new Counter('redirect_server_errors');

export default function () {
  const response = http.get(`${base}/${code}`, {
    redirects: 0, timeout: '5s', tags: { name: 'GET /{code}' },
    responseCallback: http.expectedStatuses(301),
  });
  const ok = response.status === 301 && /^https?:\/\//.test(response.headers.Location || '');
  successful.add(ok ? 1 : 0);
  failed.add(!ok);
  failures.add(ok ? 0 : 1);
  throttled.add(response.status === 429 ? 1 : 0);
  serverErrors.add(response.status >= 500 ? 1 : 0);
  check(response, { '301 with an HTTP(S) Location': () => ok });
  sleep(1); // Smoke: at most approximately 2 requests/second, including latency.
}
