import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { runInNewContext } from 'node:vm';

const code = readFileSync(new URL('../url-rewrite.js', import.meta.url), 'utf8');
const handler = runInNewContext(code + '\nhandler');
for (const [uri, expected] of [
  ['/', '/index.html'],
  ['/dashboard', '/dashboard/index.html'],
  ['/links/', '/links/index.html'],
  ['/auth/callback/', '/auth/callback/index.html'],
  ['/_next/static/chunks/app.js', '/_next/static/chunks/app.js'],
  ['/favicon.ico', '/favicon.ico'],
  ['/missing', '/missing/index.html'],
  ['/missing.png', '/missing.png'],
]) {
  test(uri, () => {
    const querystring = { code: { value: 'code' }, state: { value: 'state' } };
    const request = { uri, querystring, method: 'GET' };
    assert.deepEqual(handler({ request }), { uri: expected, querystring, method: 'GET' });
  });
}
