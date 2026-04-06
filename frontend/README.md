# Frontend (React + Vite)

This app provides a UI to create short URLs and copy them to clipboard.

## Prerequisites

- Node.js 22.x
- npm 11.x

## Environment Variables

Copy `.env.example` to `.env` and update values:

- `VITE_BASE_URL`: Public base URL where users open short links.
- `VITE_API_PROXY_TARGET`: Backend base URL used by Vite dev proxy.

## Local Development

```bash
npm install
npm run dev
```

The app runs at `http://localhost:5173` by default.

## Build

```bash
npm run build
npm run preview
```
