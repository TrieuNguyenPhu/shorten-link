# Frontend (React + Vite)

Frontend application for creating short URLs and copying the generated link.

## Prerequisites

- Node.js 22.x
- npm 11.x
- A running backend API (local SAM or deployed API Gateway)

## Environment Variables

Create `.env` from `.env.example`:

```bash
cp .env.example .env
```

Variables:

- `VITE_BASE_URL`: Public base URL used to compose short links for users.
- `VITE_API_PROXY_TARGET`: Backend target for Vite proxy (`/api/*`, `/link/*`) in development.

Current production values:

```env
VITE_BASE_URL=https://npt-shortenlink.dev
VITE_API_PROXY_TARGET=https://npt-shortenlink.dev
```

For local backend testing, you can temporarily use:

```env
VITE_BASE_URL=http://localhost:3000
VITE_API_PROXY_TARGET=http://localhost:3000
```

## Run Locally

```bash
npm install
npm run dev
```

Default URL: `http://localhost:5173`

## Build

```bash
npm run build
npm run preview
```

Build output is generated in `dist/`.

## Deploy to S3

From `frontend/dist`:

```bash
aws s3 sync . s3://<your-bucket-name>/ --delete
```

Notes:

- Bucket must already exist in the active AWS account/region.
- If `NoSuchBucket` appears, verify account/profile and bucket name.

## Serve via CloudFront (Recommended)

1. Create a CloudFront distribution with your S3 bucket as origin (use OAC).
2. Keep default behavior for static files (`GET, HEAD`).
3. Add API Gateway as a second origin.
4. Add behavior `/api/*` -> API origin.
5. Add behavior `/link/*` -> API origin.
6. Use `CachingDisabled` for API behaviors.
7. Set viewer protocol policy to `Redirect HTTP to HTTPS`.

After each deploy:

```bash
aws cloudfront create-invalidation --distribution-id <distribution-id> --paths "/*"
```

Current distribution ID example:

```bash
aws cloudfront create-invalidation --distribution-id E3JILVKN0E9ZVK --paths "/*"
```

## Verification Checklist

1. Open frontend URL and submit a long URL.
2. Confirm API returns a short code.
3. Confirm generated short URL opens and redirects to original URL.
4. Confirm copy-to-clipboard works.
