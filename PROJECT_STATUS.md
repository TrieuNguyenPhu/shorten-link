# PROJECT_STATUS.md — Trạng Thái Hiện Tại Của Dự Án

> **Cập nhật lần cuối:** 2026-10-26 (Sau khi hoàn thành **Phase 0 / Task 0.1**)
> **Quy tắc bắt buộc:** File này luôn được cập nhật ngay sau khi hoàn thành mỗi Task trước khi mở Pull Request tương ứng.

---

## 1. Tổng Quan Trạng Thái Hiện Tại

| Hạng mục | Trạng thái | Chi tiết |
| --- | --- | --- |
| **Giai đoạn hiện tại (Current Phase)** | **Phase 0 — Hoàn tất (`COMPLETE`)** | Đã hoàn thành kiểm toán bản gốc, dọn sạch AWS & local, cài đặt 13 bộ skills, thiết lập `AGENTS.md` và kế hoạch chi tiết |
| **Task vừa hoàn thành** | **Task 0.1** | Clean Reset, 13 Skill Packs Setup, `AGENTS.md`, Reference Audit & Master Plan |
| **Task tiếp theo (Next Up)** | **Task 1.1 (Phase 1)** | Khởi tạo cấu trúc Monorepo, `Makefile`, Go module (`backend/`), Domain Models, Base62 Generator & URL Validator kèm unit tests |
| **Trạng thái AWS Cloud** | **Clean (0 tài nguyên)** | Đã xóa toàn bộ 64 resources ở stack `dev` cũ và 5 resources ở stack `bootstrap` cũ; không phát sinh chi phí |
| **13 Bộ Skills & `AGENTS.md`** | **Active (`314` skills)** | Đã cài đặt tại `~/.agents/skills` và liên kết qua `.agents/skills.json` + `~/.gemini/config/skills.json` |

---

## 2. Bảng Theo Dõi Tiến Độ Từng Phase & Pull Request

| Phase | Task | Nội dung công việc | Nhánh (Branch) | Trạng thái | Pull Request |
| --- | --- | --- | --- | --- | --- |
| **Phase 0** | **Task 0.1** | Dọn sạch AWS & local, cài đặt 13 bộ skills, `AGENTS.md`, Reference Audit, Architecture & Master Plan | `chore/phase-00-reset-and-architecture-plan` | ✅ **Completed** | [PR #65](https://github.com/TrieuNguyenPhu/shorten-link/pull/65) |
| **Phase 1** | **Task 1.1** | Monorepo skeleton, `Makefile`, Go module, Domain Models, Base62 Generator & URL Validator + Unit Tests | `feat/phase-01-task-01-monorepo-and-domain` | ⏳ Pending | — |
| **Phase 1** | **Task 1.2** | Observability Middleware (`slog` JSON, CloudWatch EMF, X-Ray) & Script đóng gói Lambda `arm64` ZIP | `feat/phase-01-task-02-observability-and-packaging` | ⏳ Pending | — |
| **Phase 2** | **Task 2.1** | DynamoDB Single-Table Repository, Link Service & 4 Go HTTP Lambdas (`create`, `resolve`, `list`, `delete`) + Tests | `feat/phase-02-task-01-core-crud-lambdas` | ⏳ Pending | — |
| **Phase 2** | **Task 2.2** | Terraform Bootstrap (`S3` state + lockfile) & Core Modules (`dynamodb`, `cognito`, `lambda`, `http_api`) | `feat/phase-02-task-02-terraform-core-infra` | ⏳ Pending | — |
| **Phase 3** | **Task 3.1** | Go Lambda `click-aggregator` (`TransactWriteItems`, `CancellationReasons`, `ReportBatchItemFailures`) + Tests | `feat/phase-03-task-01-click-aggregator-lambda` | ⏳ Pending | — |
| **Phase 3** | **Task 3.2** | Terraform Module `event_pipeline` (EventBridge custom bus, rule, SQS `ClickQueue` + `ClickDLQ`, ESM) | `feat/phase-03-task-02-terraform-event-pipeline` | ⏳ Pending | — |
| **Phase 4** | **Task 4.1** | Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher (`410 Gone`) & `moderate-check` + ASL | `feat/phase-04-task-01-websocket-and-moderation-lambdas` | ⏳ Pending | — |
| **Phase 4** | **Task 4.2** | Terraform Modules `websocket_api` (`$connect`, `$disconnect`, stage `prod`) & `moderation_sfn` | `feat/phase-04-task-02-terraform-websocket-and-sfn` | ⏳ Pending | — |
| **Phase 5** | **Task 5.1** | Next.js App Router (`output: "export"`), Cognito PKCE Auth & trang Landing, Login, Auth Callback | `feat/phase-05-task-01-nextjs-auth-and-pages` | ⏳ Pending | — |
| **Phase 5** | **Task 5.2** | Trang Dashboard & My Links, tích hợp HTTP API thật & Native WebSocket Hook (`useRealtimeClicks`) | `feat/phase-05-task-02-dashboard-api-and-websocket` | ⏳ Pending | — |
| **Phase 5** | **Task 5.3** | Terraform Module `frontend_hosting` (Private S3 + CloudFront OAC + Cache Policies + URL Rewrite) | `feat/phase-05-task-03-terraform-s3-cloudfront-oac` | ⏳ Pending | — |
| **Phase 6** | **Task 6.1** | Terraform Modules `observability` (Alarms, SNS, Dashboard, X-Ray) & `codedeploy_canary` (`live` alias) | `feat/phase-06-task-01-observability-and-canary` | ⏳ Pending | — |
| **Phase 6** | **Task 6.2** | GitHub Actions CI/CD (AWS OIDC, Canary Deploy, S3/CloudFront Sync), k6 Load Test & Runbook E2E | `feat/phase-06-task-02-cicd-k6-and-runbook` | ⏳ Pending | — |

---

## 3. Tài Liệu Cốt Lõi Đã Khởi Tạo Ở Phase 0

- [`AGENTS.md`](./AGENTS.md): Quy chuẩn làm việc, kiến trúc bắt buộc, tích hợp 13 bộ skills và quy trình PR theo từng task.
- [`docs/phase-0-reference-audit.md`](./docs/phase-0-reference-audit.md): Phân tích chi tiết 21 bài viết và toàn bộ mã nguồn repo tham chiếu `nghiadaulau/serverless-url-shortener-aws`.
- [`docs/architecture-and-contracts.md`](./docs/architecture-and-contracts.md): Sơ đồ kiến trúc Mermaid, hợp đồng API HTTP/WebSocket, thiết kế DynamoDB Single-Table & giải thích 9 access patterns.
- [`docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`](./docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md): Kế hoạch triển khai chi tiết từng Phase (Phase 0 → Phase 6) và từng Task.
- [`task_plan.md`](./task_plan.md), [`findings.md`](./findings.md), [`progress.md`](./progress.md): Hệ thống file quản lý bộ nhớ làm việc theo chuẩn `planning-with-files`.
