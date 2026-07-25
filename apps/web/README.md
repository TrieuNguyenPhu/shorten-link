# NPT ShortenLink Web

Frontend Next.js App Router cho `npt-shortenlink.dev`. Local dùng `NEXT_PUBLIC_API_BASE_URL` từ `.env.local`; production gọi `/api/*` cùng origin qua CloudFront nên không cần biến này.

## Chạy local

Từ root repository:

```powershell
pnpm install --frozen-lockfile
pnpm dev:web
```

Mở `http://localhost:3000`. Backend phải chạy tại `http://localhost:8080` bằng `pnpm dev:api` trong terminal khác.

## Kiểm chứng và static export

```powershell
pnpm lint:web
pnpm build:web
```

Build production sinh static site tại `apps/web/out` để upload lên S3/CloudFront và luôn dùng `/api/*` cùng origin. `NEXT_PUBLIC_API_BASE_URL` chỉ override API trong development; nếu đổi production sang API khác origin, cần cập nhật client và CSP một cách chủ đích.
