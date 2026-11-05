# Serverless URL Shortener

Hệ thống rút gọn URL có analytics theo thời gian thực, xây dựng bằng Go Lambda, DynamoDB, EventBridge, SQS/DLQ, API Gateway HTTP/WebSocket, Step Functions, Cognito Google và Terraform. Frontend dùng Next.js static export, phục vụ từ S3 private qua CloudFront OAC.

## Mục tiêu sử dụng

Clone repository, tạo đầu vào Terraform, chạy bootstrap rồi triển khai môi trường `dev`. Sau khi cấu hình domain, Google OAuth callback và output API/CloudFront cho frontend, hệ thống cung cấp tạo/list/xóa/redirect link, analytics bất đồng bộ, cập nhật WebSocket sau commit và moderation qua Step Functions.

## Trạng thái hiện tại

Backend Go, các Lambda, DynamoDB, HTTP API, EventBridge, SQS/DLQ, WebSocket và Step Functions đã có mã nguồn cùng kiểm thử local. Frontend đã có Trang chủ, đăng nhập Google PKCE, Dashboard realtime và Kiến trúc dự án; được xuất tĩnh lên S3 private + CloudFront OAC. Observability nâng cao, canary và CD tự động thuộc Phase 6 chưa hoàn thành. Repository hiện chưa đạt điều kiện “clone rồi `terraform apply` là toàn bộ tính năng production hoạt động”.

## Bắt đầu

Yêu cầu: Go theo `backend/go.mod`, Node/pnpm theo `frontend/package.json`, Terraform 1.17.1 và AWS credentials có quyền triển khai.

```sh
git clone https://github.com/TrieuNguyenPhu/shorten-link.git
cd shorten-link
bash scripts/build-lambdas.sh
terraform -chdir=infrastructure/terraform/bootstrap init
terraform -chdir=infrastructure/terraform/bootstrap apply
terraform -chdir=infrastructure/terraform/environments/dev init -backend-config=backend.hcl
terraform -chdir=infrastructure/terraform/environments/dev apply
```

Dùng `*.tfvars.example` làm mẫu. Secret Google OAuth truyền qua biến môi trường hoặc secret manager; không commit secret hay state.

## Kiểm tra

```sh
go -C backend test -race ./...
go -C backend vet ./...
corepack pnpm@10.33.2 --dir frontend install --frozen-lockfile
corepack pnpm@10.33.2 --dir frontend lint
corepack pnpm@10.33.2 --dir frontend typecheck
corepack pnpm@10.33.2 --dir frontend build
corepack pnpm@10.33.2 --dir frontend test
corepack pnpm@10.33.2 --dir frontend audit --prod
```

Chi tiết vận hành nằm trong [`docs/operations.md`](docs/operations.md). Trạng thái triển khai tại [`PROJECT_STATUS.md`](PROJECT_STATUS.md). Quy tắc repository tại [`AGENTS.md`](AGENTS.md).

Website hiện tại: [Shrnk](https://d2aamyg2mt28wl.cloudfront.net). Public config local và cách chạy test tích hợp tại [docs/local-testing.md](docs/local-testing.md).

Kiến trúc và schema: [docs/architecture-and-contracts.md](docs/architecture-and-contracts.md). Nguồn tham khảo: [series serverless AWS](https://kkloudtarus.net/blog/series/serverless-on-aws-event-driven) và [repository gốc](https://github.com/nghiadaulau/serverless-url-shortener-aws).

## Phase 6: vận hành và phát hành

Xem [runbook CI/CD và E2E](docs/e2e-verification-and-operations.md). CD chỉ manual từ main qua protected environment `dev`: tạo saved plan → review → apply đúng plan → build frontend → canary thành công → S3/CloudFront. Merge/provision Task 6.1 ([PR #89](https://github.com/TrieuNguyenPhu/shorten-link/pull/89)) trước sử dụng. Chưa apply Phase 6 hoặc kiểm chứng canary thật trên AWS.

`make check` gồm Go race/vet, build tám ZIP, Terraform fmt/init/validate/mock tests, frontend lint/typecheck/build/tests và release/k6 contract tests. `DYNAMODB_ENDPOINT=<local-url> make test-integration` kiểm repository/HTTP/WebSocket với DynamoDB Local; AWS smoke/load test phải được cấp phép riêng.
