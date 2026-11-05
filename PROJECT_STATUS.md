# Trạng thái dự án

Cập nhật: 2026-11-05.

## Đang thực hiện

- Frontend refresh Shrnk đã hoàn tất ở mức implementation/local: branding, icon link, auth-aware Navbar, Source Code, architecture route, landing create form và dashboard chỉ đọc.

- Phase 5 — `in_progress`.
- Task 5.1 — `complete (implementation/local)`: Next.js static landing, Google-only Cognito PKCE và các trang landing/login/callback theo Doodle & Sketch UI đã hoàn thiện trên nhánh `feat/phase-05-task-01-nextjs-auth-and-pages`.

### Bằng chứng Task 5.1

- `pnpm lint`, `pnpm typecheck`, `pnpm test` (12 tests), `pnpm build` và artifact tests đều PASS trong `frontend/`.
- Build tạo `out/index.html`, `out/login/index.html`, `out/auth/callback/index.html`, `out/dashboard/index.html`, `out/404.html` và static assets.
- Auth kiểm `state`, PKCE S256, `id_token.nonce`, expiry, URL allowlist và dọn query callback; không có password/signup flow.
- Chưa chạy Terraform apply, CloudFront hosting hoặc Google browser federation E2E trong task local này.
- Full Go suite/vet hiện bị chặn bởi thay đổi backend có sẵn ngoài task: `backend/internal/config/config.go:27` giữ biến `dest` nhưng dòng gán cấu hình đã bị xóa; chưa sửa trong commit frontend.
- Release checkout dùng để deploy đã chạy `go test -race ./...` và `go vet ./...` PASS. Lỗi trên chỉ còn ở working tree cũ có các thay đổi backend chưa commit.

## Phase 5 đã triển khai dev AWS

- Task 5.2 đã hoàn thiện ở commit `babf5af` trên branch `feat/phase-05-task-02-dashboard-api-and-websocket`: dashboard/My Links, API client cursor pagination, CRUD thật và WebSocket reconnect/max merge.
- Task 5.3 đã triển khai từ commit `ea53ccd` trên branch `feat/phase-05-task-03-terraform-s3-cloudfront-oac`, với publisher pin fix `0c624b0`: S3 private, OAC, CloudFront Function rewrite, cache policies và publisher static.
- PRs: [Task 5.1 #79](https://github.com/TrieuNguyenPhu/shorten-link/pull/79), [Task 5.3 #80](https://github.com/TrieuNguyenPhu/shorten-link/pull/80). Task 5.2 đang chờ publish PR.
- AWS dev `ap-southeast-1`: CloudFront `https://d2aamyg2mt28wl.cloudfront.net`, API `https://txnaexyh3d.execute-api.ap-southeast-1.amazonaws.com`, WebSocket `wss://gshal4m5z6.execute-api.ap-southeast-1.amazonaws.com/prod`.
- Smoke AWS PASS: create/list/delete Lambda owner flow, HTTP 301/404, EventBridge → SQS → aggregator, WebSocket push `clicks=1`, CloudFront routes/assets/missing objects, browser mobile/desktop signed-out Google redirect và callback redaction.
- Google federation E2E cần tài khoản Google thử nghiệm trong browser. AWS access-token E2E chưa chạy vì môi trường không có `E2E_ACCESS_TOKEN_A`.

## Đã có

- Go Lambda arm64 cho create/resolve/list/delete, click aggregator, WebSocket connect/disconnect và moderation check.
- DynamoDB single-table, HTTP API JWT, EventBridge, SQS + DLQ, WebSocket API và Step Functions Terraform.
- Unit, race, vet, DynamoDB Local và HTTP/WebSocket local tests.
- Cognito Google-only và remote state bootstrap Terraform.

## Còn thiếu để vận hành production

- Hoàn thiện dashboard Next.js và luồng API/WebSocket thật.
- Terraform S3 private + CloudFront OAC cho frontend và custom domain/certificate.
- CloudWatch alarms/dashboard, SNS, X-Ray và CodeDeploy canary.
- CI/CD build, deploy, invalidation CloudFront và AWS E2E.

## Nguyên tắc triển khai

Không chạy `terraform apply`, load test hoặc thao tác AWS ngoài kiểm tra được ủy quyền. Trước deploy cần build ZIP Lambda, tạo bootstrap state/OIDC, điền Google OAuth và domain URLs, sau đó apply bootstrap rồi environment `dev`. Không commit secret, `.tfstate` hoặc file `.tfvars` thật.
