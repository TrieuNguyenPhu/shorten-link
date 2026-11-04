# Frontend tĩnh — Next.js App Router

Scaffold dùng Next.js16.4.3, TypeScript và Tailwind CSS. Cấu hình output export, trailingSlash và ảnh unoptimized tạo HTML/CSS/JS cho S3 private + CloudFront; không cần Node.js runtime khi phục vụ.

## Kiểm tra local

Yêu cầu Node26.13.0 theo CI và Corepack để chọn pnpm10.33.2. Chạy từ thư mục frontend:

```sh
corepack pnpm@10.33.2 install --frozen-lockfile
corepack pnpm@10.33.2 lint
corepack pnpm@10.33.2 typecheck
corepack pnpm@10.33.2 build
corepack pnpm@10.33.2 test
corepack pnpm@10.33.2 audit --prod
```

Build tạo out/index.html, out/404.html và asset tĩnh. Chạy build trước test; Node test runner kiểm HTML/404, các asset tham chiếu tồn tại và không cần image server. Typecheck tự sinh route types trước tsc. CI chạy cùng các bước trên; không dùng next start để phục vụ bản export.

Phát triển local bằng `corepack pnpm@10.33.2 dev`. Font Google được tải lúc build nên máy build cần mạng; hosting chỉ phục vụ artifact xuất ra.

## Phạm vi hiện tại

Đây là scaffold đã kiểm tra static export. Google qua Cognito PKCE, landing/login/callback theo Doodle UI, dashboard, API và WebSocket thật vẫn thuộc Task5.1/5.2. Chưa triển khai hosting AWS hoặc E2E đăng nhập.
