# Shorten Link

A full-stack URL shortener project with:

- `frontend`: React + Vite web client
- `backend`: AWS SAM serverless API (Lambda + API Gateway + DynamoDB)

## Architecture

1. User submits a long URL from the web client.
2. Frontend sends `POST /api/generate-short-url` to backend.
3. Backend stores `{short_url, original_url}` in DynamoDB.
4. User opens `/link/{short_url}` and backend returns an HTTP redirect.

## Repository Structure

```text
shorten-link/
├─ frontend/    # React app
└─ backend/     # AWS SAM application
```

## Prerequisites

- Node.js 22.x and npm 11.x
- Python 3.12.x
- AWS CLI configured with valid credentials
- AWS SAM CLI 1.160.1

Version pin files:

- `.nvmrc` -> `22.13.1`
- `backend/.python-version` -> `3.12.4`

## Quick Start (Local Development)

### 1) Backend

```bash
cd backend
pip install -r requirements.txt
sam build
sam local start-api
```

Backend local URL is usually `http://127.0.0.1:3000`.

### 2) Frontend

```bash
cd frontend
npm install
npm run dev
```

Create `frontend/.env` from `frontend/.env.example` before running the app.

Update `frontend/.env`:

- `VITE_BASE_URL=http://localhost:3000`
- `VITE_API_PROXY_TARGET=http://localhost:3000`

Frontend runs at `http://localhost:5173`.

## Deploy Backend (AWS)

```bash
cd backend
sam build
sam deploy --guided
```

After deployment, read outputs from CloudFormation/SAM for API endpoint URLs.

## Build Frontend for Production

```bash
cd frontend
npm run build
```

Deploy `frontend/dist` to your static hosting platform (S3/CloudFront, Netlify, Vercel, etc.).

Before building for production, set:

- `VITE_BASE_URL=https://<your-public-domain>`
- `VITE_API_PROXY_TARGET=https://<your-api-domain-or-stage-base-url>`

## Testing

### Backend tests

```bash
cd backend
pip install -r tests/requirements.txt
pytest tests
```

`backend/tests/integration/test_api_gateway.py` is a live test and requires:

- `API_BASE_URL=https://<deployed-api-base-url>`

## Notes

- This repository intentionally contains only generic configuration values.
- Do not commit real secrets or personal domains into `.env` files.
