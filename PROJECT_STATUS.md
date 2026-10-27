# PROJECT_STATUS.md — Trạng Thái Hiện Tại Của Dự Án

> **Cập nhật lần cuối:** 2026-10-27 (**Phase 1 / Task 1.1 complete; Task 1.2 tiếp theo**)
> **Quy tắc bắt buộc:** File này luôn được cập nhật ngay sau khi hoàn thành mỗi Task trước khi mở Pull Request tương ứng.

---

## 1. Tổng Quan Trạng Thái Hiện Tại

| Hạng mục | Trạng thái | Chi tiết |
| --- | --- | --- |
| **Giai đoạn hiện tại (Current Phase)** | **Phase 1 — `in_progress`** | Task 1.1 hoàn tất nền tảng domain/config/Go CI; Task 1.2 tiếp theo |
| **Task hiện tại** | **Task 1.1 — `complete`** | Domain/config có unit tests, race/vet PASS và review độc lập; chờ PR review/merge |
| **Task tiếp theo (Next Up)** | **Task 1.2 (Phase 1)** | Structured logs, EMF, X-Ray và script build ZIP arm64 |
| **Trạng thái AWS Cloud** | **Đã dọn theo xác nhận người dùng** | Người dùng xác nhận ngày 2026-10-26 đã xóa toàn bộ tài nguyên triển khai trước đó; phiên hoàn thiện Phase 0 không kiểm kê AWS hay xác nhận hóa đơn |
| **13 Bộ Skills & `AGENTS.md`** | **Đã kiểm tra file/config** | Đủ 13 file skill đại diện; 314 thư mục cấp đầu có `SKILL.md`. Có cấu hình trỏ `~/.agents/skills`; không suy ra mọi host đều tự nạp |

---

## 2. Bảng Theo Dõi Tiến Độ Từng Phase & Pull Request

| Phase | Task | Nội dung công việc | Nhánh (Branch) | Trạng thái | Pull Request |
| --- | --- | --- | --- | --- | --- |
| **Phase 0** | **Task 0.1** | Xác minh setup skills/`AGENTS.md`, hoàn thiện Reference Audit, Architecture & Master Plan | `chore/phase-00-reset-and-architecture-plan` | **complete (PR chờ merge)** | [PR #65](https://github.com/TrieuNguyenPhu/shorten-link/pull/65) |
| **Phase 1** | **Task 1.1** | Monorepo, `Makefile`, Go module, Domain Models, Base62 Generator & URL Validator + Unit Tests, CI nền | `feat/phase-01-task-01-monorepo-and-domain` | complete | [PR #66](https://github.com/TrieuNguyenPhu/shorten-link/pull/66) |
| **Phase 1** | **Task 1.2** | Observability Middleware (`slog` JSON, CloudWatch EMF, X-Ray) & Script đóng gói Lambda `arm64` ZIP | `feat/phase-01-task-02-observability-and-packaging` | ⏳ Pending | — |
| **Phase 2** | **Task 2.1** | DynamoDB Repository, Link Service, EventBridge publisher, local HTTP harness & 4 HTTP Lambdas + Tests | `feat/phase-02-task-01-core-crud-lambdas` | ⏳ Pending | — |
| **Phase 2** | **Task 2.2** | Terraform Bootstrap (S3 state/lockfile, OIDC) & Core Modules, Cognito Google-only, custom event bus | `feat/phase-02-task-02-terraform-core-infra` | ⏳ Pending | — |
| **Phase 3** | **Task 3.1** | Go Lambda `click-aggregator` (`TransactWriteItems`, `CancellationReasons`, `ReportBatchItemFailures`) + Tests | `feat/phase-03-task-01-click-aggregator-lambda` | ⏳ Pending | — |
| **Phase 3** | **Task 3.2** | Terraform Module `event_pipeline` (nhận bus từ Task 2.2, tạo rule, SQS `ClickQueue` + `ClickDLQ`, ESM) | `feat/phase-03-task-02-terraform-event-pipeline` | ⏳ Pending | — |
| **Phase 4** | **Task 4.1** | Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher (`410 Gone`) & `moderate-check` + ASL | `feat/phase-04-task-01-websocket-and-moderation-lambdas` | ⏳ Pending | — |
| **Phase 4** | **Task 4.2** | Terraform Modules `websocket_api` (`$connect`, `$disconnect`, stage `prod`) & `moderation_sfn` | `feat/phase-04-task-02-terraform-websocket-and-sfn` | ⏳ Pending | — |
| **Phase 5** | **Task 5.1** | Next.js static, Google-only qua Cognito PKCE, Landing/Login/Callback theo Doodle & Sketch UI | `feat/phase-05-task-01-nextjs-auth-and-pages` | ⏳ Pending | — |
| **Phase 5** | **Task 5.2** | Dashboard & My Links Doodle & Sketch, API thật & Native WebSocket Hook (`useRealtimeClicks`) | `feat/phase-05-task-02-dashboard-api-and-websocket` | ⏳ Pending | — |
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

## 4. Bằng Chứng Kiểm Tra Và Giới Hạn

- Tài liệu đã đối chiếu source ở commit `a5ffed9780d0447fc75a42def3693732bdc87907` (21 tracked files) và đủ 21 bài qua cả hai trang series; các giới hạn nguồn được ghi trong audit.
- Kiểm tra Phase 0: `git diff --check`, liên kết Markdown nội bộ, JSON examples, số bài và file nguồn. Không chạy `go test`, `pnpm build` hay `terraform validate` vì các dự án thực thi chỉ bắt đầu từ Phase 1–2.
- E2E Google → Cognito → CloudFront/API, resource inventory và chi phí AWS chưa xác minh; Google OAuth Web client/secret là đầu vào bên ngoài cần chuẩn bị ở Task 2.2, không có secret trong repository.
- Thư mục `frontend/` xuất hiện dưới dạng file **chưa được Git theo dõi** trong lúc hoàn thiện tài liệu; đây là scaffold Next.js chưa thuộc Task 0.1/PR #65. Giữ nguyên các file đó, chưa xác nhận build hoặc tính năng.

## 5. Bằng chứng Phase 1 / Task 1.1
- Go 1.26.5: test/vet trên Windows PASS; `GOTOOLCHAIN=go1.26.5 go -C backend test -race -v ./...` và vet trên WSL Ubuntu/GCC PASS.
- `gofmt -l backend` không có file; `git diff --check` PASS. Review độc lập domain/config/CI không có finding cần sửa.
- Test-first đã xác nhận RED do các API chưa tồn tại, sau đó GREEN. Coverage config 100%, domain 88.2%; không suy ra bảo đảm nghiệp vụ chưa triển khai.
- PR phụ thuộc #65; frontend untracked được giữ nguyên. Chưa gọi AWS hoặc chạy Terraform/pnpm.
- Cập nhật review Task1.1: minimum Go/CI chuyển sang1.26.10 để khắc phục stdlib vulnerabilities phát hiện trong Phase1; bằng chứng1.26.5 bên trên là lịch sử.
