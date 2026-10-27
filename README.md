# Serverless URL Shortener with Realtime Analytics (AWS + Go + Terraform + Next.js)

Đặc tả và kế hoạch tái triển khai hệ thống **Serverless URL Shortener with Realtime Analytics** trên AWS bằng **Golang (`provided.al2023`, `arm64`)**, **Terraform (HCL)**, và **Next.js App Router (`output: "export"`)** trên **Amazon S3 (Private) + CloudFront Origin Access Control (OAC)**.

Phase 0 hoàn thiện tài liệu ngày 2026-10-27, dựa trên lượt đọc nguồn ngày 2026-10-26. Chưa có ứng dụng Go/Next.js/Terraform chạy được; kiểm thử AWS và triển khai thuộc các phase tiếp theo và cần được cho phép. AWS cũ đã được xóa theo xác nhận người dùng, không phải kết quả kiểm kê AWS trong phiên này.

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
