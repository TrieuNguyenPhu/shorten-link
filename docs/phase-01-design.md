# Phase 01 design

Build a single-user local URL shortener that teaches the eventual AWS architecture through working software. No AWS resources, authentication substitutes, analytics, moderation, or deployment are included.

The browser runs a statically exported Next.js App Router application. Client components call a separate Go HTTP API. Go application use cases depend on a small repository port; a DynamoDB adapter implements that port. HTTP request types stay in the HTTP adapter so future native Lambda handlers can call the same use cases.

Each link stores its code, destination, enabled flag, UTC creation time, and optional UTC expiry. Generated codes contain 48 random bits, encoded as eight URL-safe characters. Custom aliases contain 3–32 ASCII letters, digits, underscores, or hyphens. Atomic conditional puts reserve codes. Generated collisions retry at most five times; custom collisions return 409. Destination URLs must be absolute HTTP(S) URLs without embedded credentials, whitespace, or control characters. The server never fetches destinations.

Single table: `PK=LINK#<code>`, `SK=METADATA`. `GSI1PK=LINKS`, `GSI1SK=<UTC creation time>#<code>` supports newest-first paginated listing. Reads used for resolution are strongly consistent; listing is eventually consistent. Disabled or expired redirects return 410 and missing links return 404. Redirects use 302 and no-store so status changes are respected. Expiry is enforced by the application, without relying on DynamoDB TTL deletion.

Endpoints: health, create, get information, list (25 per page), update enabled state, and redirect. JSON errors always use `{ "error": { "code": "...", "message": "..." } }`. The frontend provides creation, optional alias/expiry, link listing, copy/open actions, and enable/disable actions with loading, empty, success, and error states. Counts describe loaded links, not invented analytics.

Local safety: API binding and database endpoint must be loopback; SDK credentials are fixed local-only values and shared AWS configuration is never loaded. CORS allows exactly the configured local frontend origin. Docker exposes DynamoDB on loopback with persistent named storage. Initialization is idempotent, waits for readiness, and checks the existing schema without deleting data.

Verification: business tests cover validation, collision retry/exhaustion, expiration boundaries, and repository failures; HTTP tests cover status codes, JSON errors, CORS and redirects; DynamoDB integration tests exercise conditional/concurrent writes and pagination. Playwright exercises the built static application against the real Go API and DynamoDB Local at desktop and mobile sizes.
