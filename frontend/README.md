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

Task 5.1 đã có landing, login Google-only qua Cognito authorization code + PKCE S256, callback có state/nonce/expiry validation và route dashboard handoff. UI dùng phong cách Doodle & Sketch, không có password/signup flow. Dashboard/My Links gọi API thật và WebSocket realtime thuộc Task 5.2; S3 private + CloudFront OAC thuộc Task 5.3. Chưa chạy browser federation E2E hoặc hosting AWS trong local task này.

Public build cần sáu biến `NEXT_PUBLIC_*` từ Terraform outputs hoặc môi trường local. API/CloudFront/Cognito URL production phải dùng HTTPS/WSS và không được là placeholder; callback phải kết thúc bằng `/auth/callback/`.

## Cấu hình Google login local

Tạo `frontend/.env.local` dựa trên `.env.example`, thay giá trị mẫu bằng các Terraform outputs sau của environment `dev`:

| Biến frontend | Terraform output |
| --- | --- |
| `NEXT_PUBLIC_API_BASE_URL` | `http_api_endpoint` |
| `NEXT_PUBLIC_WS_URL` | `websocket_endpoint` (giữ đường dẫn stage `/prod`) |
| `NEXT_PUBLIC_COGNITO_DOMAIN` | `cognito_login_domain` |
| `NEXT_PUBLIC_COGNITO_CLIENT_ID` | `cognito_client_id` |

Giữ callback `http://localhost:3000/auth/callback/` và logout `http://localhost:3000/` khi dùng local; cả hai phải được đăng ký đúng trong Cognito. Khởi động lại `pnpm dev` sau khi cập nhật env. Khi xuất bản CloudFront, dùng origin CloudFront thay localhost và build lại vì `NEXT_PUBLIC_*` được ghi vào JavaScript lúc build; `scripts/publish-frontend.sh` lấy các giá trị này từ Terraform outputs.

File JSON Google OAuth chứa Google client ID/secret để cấu hình Cognito IdP, không phải public Cognito app client ID của frontend. Không chép client secret vào `NEXT_PUBLIC_*`, public assets hoặc Git. `.env.local` chỉ chứa sáu giá trị public và được Git bỏ qua.
