# Serverless URL Shortener with Realtime Analytics (AWS + Go + Terraform + Next.js)

Đặc tả và kế hoạch tái triển khai hệ thống **Serverless URL Shortener with Realtime Analytics** trên AWS bằng **Golang (`provided.al2023`, `arm64`)**, **Terraform (HCL)**, và **Next.js App Router (`output: "export"`)** trên **Amazon S3 (Private) + CloudFront Origin Access Control (OAC)**.

Phase 1 cung cấp nền tảng Go chạy local; HTTP handlers, frontend và Terraform thuộc các phase sau. AWS cũ đã được xóa theo xác nhận người dùng; phiên này không triển khai AWS.

Dự án bám sát kiến trúc hướng sự kiện, thiết kế bảng đơn DynamoDB và luồng xử lý của:
- **Chuỗi 21 bài viết:** [Serverless Thực Chiến Trên AWS](https://kkloudtarus.net/blog/series/serverless-on-aws-event-driven)
- **Repository gốc:** [nghiadaulau/serverless-url-shortener-aws](https://github.com/nghiadaulau/serverless-url-shortener-aws)

---

## 1. Bảng Đối Chiếu Thay Đổi Kiến Trúc

| Thành phần | Bản gốc | Bản triển khai này |
| --- | --- | --- |
| **Lambda Runtime** | Node.js 22 / TypeScript (`esbuild`) | **Golang** (`provided.al2023`, `arm64`, binary `bootstrap`) |
| **Infrastructure as Code** | AWS SAM (`template.yaml`) | **Terraform (HCL)** quản lý 100% hạ tầng |
| **Frontend Dashboard** | `web/index.html` (Vanilla HTML/JS) | **Next.js App Router + TypeScript + Tailwind CSS** (`output: "export"`) |
| **Frontend Hosting** | File tĩnh cục bộ | **Amazon S3 (Private) + CloudFront (OAC)** |
| **Xác thực** | Cognito email/mật khẩu trong web gốc | **Chỉ Google qua Cognito User Pool, Authorization Code + PKCE** |
| **Phong cách UI** | HTML/JS tối giản | **Hand-Drawn / Doodle & Sketch UI**, responsive, dễ đọc |
| **Dịch vụ Backend AWS** | API Gateway (HTTP + WebSocket), DynamoDB Single-Table (`PK`/`SK` + Sparse `GSI1` + `ttl`), Cognito JWT, EventBridge, SQS + DLQ, Step Functions, CloudWatch/X-Ray, SNS, CodeDeploy Canary | **Giữ dịch vụ, schema và access patterns; sửa lỗi publish, transaction và notifier theo đặc tả** |

---

## 2. Tài Liệu & Trạng Thái Dự Án

- 📊 **Trạng thái hiện tại của dự án:** [`PROJECT_STATUS.md`](./PROJECT_STATUS.md)
- 🤖 **Quy chuẩn Agent & 13 Bộ Skills:** [`AGENTS.md`](./AGENTS.md)
- 🔍 **Báo cáo Audit bản gốc (21 bài + source code):** [`docs/phase-0-reference-audit.md`](./docs/phase-0-reference-audit.md)
- 🏗️ **Kiến trúc Mermaid, Hợp đồng API & DynamoDB Single-Table:** [`docs/architecture-and-contracts.md`](./docs/architecture-and-contracts.md)
- 🗺️ **Kế hoạch triển khai chi tiết (Phase 0 → Phase 6):** [`docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`](./docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md)
- 📝 **Bộ nhớ làm việc (`planning-with-files`):** [`task_plan.md`](./task_plan.md) · [`findings.md`](./findings.md) · [`progress.md`](./progress.md)

---

## 3. Lộ Trình Phát Triển (Development Phases)

1. **Phase 0 — Reference Audit & Architecture** *(Hoàn tất)*
2. **Phase 1 — Monorepo & Local Foundation** *(Task 1.1 & Task 1.2)*
3. **Phase 2 — Core URL Shortener** *(Task 2.1 & Task 2.2)*
4. **Phase 3 — Event-Driven Analytics** *(Task 3.1 & Task 3.2)*
5. **Phase 4 — Realtime WebSocket & Moderation** *(Task 4.1 & Task 4.2)*
6. **Phase 5 — Frontend Next.js & S3/CloudFront Hosting** *(Task 5.1, Task 5.2 & Task 5.3)*
7. **Phase 6 — Production Operations, Observability & Canary CI/CD** *(Task 6.1 & Task 6.2)*

## 4. Kiểm tra nền tảng local

Yêu cầu Go 1.26.5 (pin trong `backend/go.mod` và CI). Chạy từ root:

```sh
go -C backend test -race ./...
go -C backend vet ./...
# Hoặc: make check
```

Race detector cần CGO và C compiler trên máy chạy test. CI dùng Linux; trên Windows có thể chạy qua WSL với Go/GCC. `gofmt -l backend` phải không liệt kê file nào.

`domain.GenerateShortCode(7)` dùng crypto/rand, `domain.NormalizeURL` nhận HTTP/HTTPS absolute, IP/localhost và DNS ASCII (IDN dùng punycode), không truy cập đích URL. `FormatCreatedAt` trả UTC cố định 3 chữ số milliseconds để sắp xếp GSI1.

`config.Load()` đọc `TABLE_NAME` bắt buộc; `EVENT_BUS`, `WS_ENDPOINT` và `PUBLIC_REDIRECT_BASE_URL` tùy chọn theo Lambda. Caller kiểm tra biến cần cho nghiệp vụ khi khởi tạo. `WS_ENDPOINT` là HTTP(S) Management API (ví dụ `https://abc.execute-api.ap-southeast-1.amazonaws.com/prod`), không phải địa chỉ `wss://` của browser. Base URL bỏ dấu `/` cuối, không nhận credentials/query/fragment; hỗ trợ path stage của API. Không đưa secret vào repository.
