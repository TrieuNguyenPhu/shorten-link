# Task Plan: Serverless URL Shortener with Realtime Analytics (Go + Terraform + Next.js)

## Goal
Tái hiện hoàn chỉnh hệ thống **Serverless URL Shortener with Realtime Analytics** trên AWS theo chuỗi 21 bài và repository `nghiadaulau/serverless-url-shortener-aws`, chuyển đổi sang **Golang (`provided.al2023`, `arm64`)**, **Terraform (HCL)**, và **Next.js Static Export (`S3 + CloudFront OAC`)**, với mỗi Task hoàn thành được tạo một **Pull Request riêng** và luôn cập nhật trạng thái hiện tại của dự án.

## Current Phase
Phase 0 — in_progress (Hoàn thiện Task 0.1 sau review PR #65)

## Phases & Tasks

### Phase 0: Reference Audit, Clean Reset & Architecture Specification
- [ ] **Task 0.1:** Hoàn thiện audit nguồn, xác minh setup skills/`AGENTS.md`, đặc tả kiến trúc và kế hoạch chi tiết sau review PR #65. AWS cũ đã được người dùng xác nhận dọn sạch; không lặp lại thao tác xóa.
- **Status:** in_progress

### Phase 1: Monorepo & Local Foundation
- [ ] **Task 1.1:** Khởi tạo cấu trúc Monorepo, `Makefile`, Go module (`backend/`), Domain Models, Base62 Generator (`crypto/rand`) & URL Validator kèm unit tests và CI nền.
- [ ] **Task 1.2:** Xây dựng Observability Middleware (`slog` JSON, CloudWatch EMF metrics, X-Ray wrapper) & script đóng gói Lambda `bootstrap` ZIP (`arm64`) kèm tests.
- **Status:** pending

### Phase 2: Core URL Shortener (DynamoDB, Cognito JWT & 4 HTTP Handlers)
- [ ] **Task 2.1:** Triển khai DynamoDB Repository, Link Service, EventBridge publisher, local HTTP harness & 4 Go HTTP Lambda Handlers kèm unit/integration tests (multi-tenant, anti-IDOR, collision retry, 301, lỗi publish).
- [ ] **Task 2.2:** Xây dựng Terraform Bootstrap (S3 remote state + lockfile, OIDC provider/deployment role) & Core Infrastructure (`dynamodb`, `cognito`, `lambda`, `http_api`, custom event bus) + `environments/dev`.
- **Status:** pending

### Phase 3: Event-Driven Analytics (EventBridge, SQS + DLQ, ClickAggregator)
- [ ] **Task 3.1:** Triển khai Go Lambda `click-aggregator` (`TransactWriteItems` 3 thao tác, phân biệt chính xác `CancellationReasons`, `ReportBatchItemFailures`) kèm unit/integration tests.
- [ ] **Task 3.2:** Xây dựng Terraform Module `event_pipeline` nhận custom bus từ Task 2.2, thêm rule `link.clicked`, SQS `ClickQueue` + `ClickDLQ` 14d, Event Source Mapping.
- **Status:** pending

### Phase 4: Realtime WebSocket & Step Functions Moderation
- [ ] **Task 4.1:** Triển khai Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher (xử lý `410 GoneException`) & `moderate-check` + `statemachine/moderation.asl.json` kèm tests.
- [ ] **Task 4.2:** Xây dựng Terraform Modules `websocket_api` (`$connect`, `$disconnect`, stage `prod`) & `moderation_sfn` (Step Functions Standard + IAM).
- **Status:** pending

### Phase 5: Frontend Next.js Dashboard & S3 + CloudFront OAC Hosting
- [ ] **Task 5.1:** Khởi tạo Next.js App Router (`output: "export"`), Cognito PKCE Auth & các trang Landing (`/`), Login (`/login`), Callback (`/auth/callback`).
- [ ] **Task 5.2:** Xây dựng trang Dashboard (`/dashboard`), My Links (`/links`), tích hợp HTTP API thật & Native WebSocket Hook (`useRealtimeClicks`).
- [ ] **Task 5.3:** Xây dựng Terraform Module `frontend_hosting` (Private S3 + CloudFront OAC + Cache Policies + CloudFront Function URL rewrite).
- **Status:** pending

### Phase 6: Production Operations, Observability, Canary CI/CD & Load Testing
- [ ] **Task 6.1:** Xây dựng Terraform Modules `observability` (CloudWatch Alarms, SNS, Dashboard, X-Ray) & `codedeploy_canary` (`LambdaCanary10Percent5Minutes` trên alias `live`).
- [ ] **Task 6.2:** Thiết lập GitHub Actions CI/CD (AWS OIDC, Canary Deploy, S3/CloudFront Sync), kịch bản `tests/load/redirect.k6.js` & tài liệu vận hành E2E.
- **Status:** pending

## Key Decisions
| Decision | Rationale |
|----------|-----------|
| Tách domain CloudFront (Frontend) và API Gateway (`GET /{code}`) | Giữ nguyên route `GET /{code}` ở gốc API Gateway như bản gốc mà không xung đột với các route tĩnh của Next.js trên CloudFront. |
| Phân loại đầy đủ `CancellationReasons` theo META/STAT/marker | Không coi mọi cancellation là duplicate; phân biệt event của link đã xóa, conflict và lỗi hỗn hợp theo hợp đồng kiến trúc. |
| Cấu hình `lifecycle { ignore_changes = [function_version, routing_config] }` trên `aws_lambda_alias.live` | Cho phép CodeDeploy quản lý việc dịch chuyển phiên bản Canary mà không bị `terraform apply` ghi đè. |
| Giữ TTL marker 24 giờ và công bố giới hạn replay | Chống trùng phụ thuộc marker còn tồn tại; không khẳng định exactly-once vô thời hạn. |
| Bootstrap OIDC tách ứng dụng, build static frontend sau khi có outputs | Tài khoản sạch cần trust/role trước khi CI authenticate; static export cần endpoint thật ở thời điểm build. |

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
| `aws_dynamodb_table` / `aws_cognito_user_pool` `deletion_protection` chặn `terraform destroy` trên stack cũ | 1 | Tắt `deletion_protection` và bật `force_destroy = true` kèm `AWS_ACCOUNT_ID_ENDPOINT_MODE=disabled`, sau đó chạy `terraform destroy` xóa sạch 64 resources ở `dev` và 5 resources ở `bootstrap`. |
| Thư mục `services/lambda` và `.codegraph` bị khóa bởi tiến trình `go` / `codegraph` nền khi xóa local | 1 | Dừng tiến trình nền đang giữ khóa và xóa sạch toàn bộ file/thư mục cũ trong workspace. |
