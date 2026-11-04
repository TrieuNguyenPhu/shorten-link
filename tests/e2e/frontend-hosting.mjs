// Run after publishing frontend/out to the Terraform-provisioned distribution.
import assert from 'node:assert/strict';

const site = new URL(process.argv[2]);
assert.ok(site.protocol === 'https:' && site.pathname === '/' && !site.search && !site.hash,
  'Pass the HTTPS origin from the frontend_url Terraform output');

async function get(path) {
  return fetch(new URL(path, site), { redirect: 'manual', signal: AbortSignal.timeout(15000) });
}

let home;
for (const path of ['/', '/login', '/dashboard', '/links/', '/auth/callback/?code=test&state=test']) {
  const response = await get(path);
  assert.equal(response.status, 200, path);
  assert.match(response.headers.get('content-type') ?? '', /text\/html/i, path);
  const html = await response.text();
  assert.match(html, /<!doctype html/i, path);
  if (path === '/') home = html;
}

const asset = home.match(/(?:src|href)="(\/_next\/static\/[^"]+\.(?:js|css))"/)?.[1];
assert.ok(asset, 'Expected a hashed Next.js asset in the exported landing page');
const response = await get(asset);
assert.equal(response.status, 200, asset);
assert.match(response.headers.get('content-type') ?? '', /javascript|text\/css/i);
assert.match(response.headers.get('cache-control') ?? '', /max-age=31536000/i);
assert.match(response.headers.get('cache-control') ?? '', /immutable/i);
await response.body.cancel();

for (const path of ['/__missing-hosting-probe__', '/_next/static/__missing-hosting-probe__.js']) {
  const response = await get(path);
  assert.ok([403, 404].includes(response.status), `Unexpected status ${response.status} for ${path}`);
  await response.body.cancel();
}
console.log('CloudFront routes, callback, immutable asset and missing-object checks passed.');
