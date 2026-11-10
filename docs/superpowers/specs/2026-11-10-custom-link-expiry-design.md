# Custom link names and expiry design

## Intent

Allow an authenticated client to choose a readable short-link name and an optional lifetime while preserving the current generated-code flow. A blank name keeps the existing random seven-character code. A blank lifetime keeps the link permanently available; otherwise the client can choose 1–365 days.

## Goals

- Accept an optional custom slug using 3–32 ASCII letters, numbers, or `-`.
- Reject malformed, reserved, or already-used slugs without overwriting another link.
- Persist an optional Unix expiry timestamp and make resolution return 404 after expiry.
- Expose expiry metadata in the create/list API and show it in the dashboard.
- Keep existing links and clients compatible when the new fields are omitted.

## Non-goals

- Editing an existing link's slug or expiry.
- Reclaiming an expired slug automatically.
- Supporting Unicode, `/`, query strings, or arbitrary punctuation in slugs.

## Approach

The recommended approach is to use the custom slug as the DynamoDB link key, with conditional writes for collision handling, and store `expiresAt` on the link item. DynamoDB's existing TTL configuration can clean up expired records asynchronously, while the resolve service enforces expiry synchronously. Alternatives that keep a random internal code plus a slug mapping add another read/write path and do not improve this use case; accepting arbitrary strings would weaken URL safety and make routing ambiguous.

## Decisions

- Request body: `{ url, alias?, expiryDays? }`.
- `alias` is trimmed; empty means generated code. Valid aliases match `[A-Za-z0-9-]{3,32}` and cannot start or end with `-` or contain `--`.
- `expiryDays` is an integer from 1 through 365; omitted or blank means no expiry.
- `expiresAt` is computed from the service clock at creation time and is omitted for permanent links.
- Collision errors are surfaced as HTTP 409. Invalid input remains HTTP 400.
- The frontend preserves the existing API client contract while adding optional inputs and expiry display.

## Verification

Add backend domain/service/handler/repository tests and frontend form/API tests covering valid values, invalid ranges, collisions, expiration, permanent links, and backward-compatible requests. Run the full repository checks before pushing and use the repository's guarded GitHub Actions deployment workflow for AWS.
