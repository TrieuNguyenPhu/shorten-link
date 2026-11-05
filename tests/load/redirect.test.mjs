// Execute the k6 script with HTTP/metric boundaries supplied by the test.
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import test from 'node:test';
import assert from 'node:assert/strict';
const source = (await readFile(new URL('./redirect.k6.js', import.meta.url), 'utf8'))
  .replace(/^import .+;\n/gm, '').replace('export const options', 'globalThis.options')
  .replace('export default function ()', 'globalThis.run = function ()');
function load(env = {}, status = 301) {
  const calls = [], counts = {};
  class Metric { constructor(name) { this.name = name; } add(value) { counts[this.name] = value; } }
  const context = vm.createContext({ __ENV: { BASE_URL: 'http://127.0.0.1:8080', CODE: 'Abcd123', ...env },
    Counter: Metric, Rate: Metric, sleep: n => calls.push(['sleep', n]),
    check: (_, checks) => assert.equal(Object.values(checks)[0](), status === 301),
    http: { expectedStatuses: n => n, get: (url, options) => { calls.push([url, options]); return { status, headers: { Location: 'https://example.org' } }; } },
  });
  vm.runInContext(source, context);
  return { context, calls, counts };
}
test('default smoke is bounded and never follows redirects', () => {
  const { context, calls } = load();
  assert.equal(context.options.scenarios.smoke.vus, 2);
  assert.equal(context.options.scenarios.smoke.duration, '10s');
  context.run();
  assert.equal(calls[0][1].redirects, 0);
  assert.equal(calls[0][1].responseCallback, 301);
  assert.deepEqual(calls[1], ['sleep', 1]);
});
test('reports success, throttle and errors separately', () => {
  for (const status of [301, 429, 500, 0]) {
    const { context, counts } = load({}, status); context.run();
    assert.equal(counts.redirect_successful, status === 301 ? 1 : 0);
    assert.equal(counts.redirect_failed, status !== 301);
    assert.equal(counts.redirect_failures, status === 301 ? 0 : 1);
    assert.equal(counts.redirect_throttled, status === 429 ? 1 : 0);
    assert.equal(counts.redirect_server_errors, status >= 500 ? 1 : 0);
  }
});
test('stress requires an explicit gate; invalid config fails before HTTP', () => {
  assert.throws(() => load({ MODE: 'stress' }), /authorization/);
  assert.equal(load({ MODE: 'stress', ALLOW_STRESS: '1' }).context.options.scenarios.stress.stages[1].target, 40);
  for (const env of [{ BASE_URL: 'https://a.example/?token=x' }, { BASE_URL: 'http://remote.example' }, { CODE: '../links' }, { MODE: 'typo' }]) assert.throws(() => load(env));
});
