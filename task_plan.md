# Task Plan: Serverless URL Shortener with Realtime Analytics (Go + Terraform + Next.js)

## Phase 6 / Task 6.2 — 2026-11-05

**Status: complete (implementation/local)**. Task 6.1 đã mở [PR #89](https://github.com/TrieuNguyenPhu/shorten-link/pull/89), chưa merge. Task 6.2 trên branch riêng từ main `f5112f1`; pipeline yêu cầu merge/provision Task 6.1 trước sử dụng. Không apply/load test AWS.


## Goal
Tái hiện hoàn chỉnh hệ thống **Serverless URL Shortener with Realtime Analytics** trên AWS theo chuỗi 21 bài và repository `nghiadaulau/serverless-url-shortener-aws`, chuyển đổi sang **Golang (`provided.al2023`, `arm64`)**, **Terraform (HCL)**, và **Next.js Static Export (`S3 + CloudFront OAC`)**, với mỗi Task hoàn thành được tạo một **Pull Request riêng** và luôn cập nhật trạng thái hiện tại của dự án.

## Current Phase

- [x] **[Task 5.5 / PR #88](https://github.com/TrieuNguyenPhu/shorten-link/pull/88) — Status: complete (implementation/deployed):** UI tối giản, shared layout/menu active/hover, Lobster; đăng nhập về Trang chủ; sửa refresh/reconnect/deletion race và hiển thị link mới. Review toàn project và low-volume AWS smoke PASS; rollout cuối `I4OJCY74UTRUNCSQ430QB61MWL` Completed.

- [x] **Sửa callback state mismatch — `Status: complete (implementation/deployed)`:** giữ PKCE khi token chưa có/hết hạn, callback chỉ exchange một lần dù effect replay. 24 tests/checks PASS; đã publish và xác minh production không còn xóa state trước token exchange trong phiên chẩn đoán.

- [x] **Publish frontend — `Status: complete`:** PR #85 (`2253a3c`) đã build với config CloudFront thật, upload S3, invalidation Completed. Hosting smoke/18 tests/title/favicon/HTML artifact comparison và Google redirect trên website thật PASS.

- [x] **Sửa Google login và metadata — `Status: complete (implementation/local)`:** public env local từ Terraform outputs, WebSocket giữ `/prod`, bỏ chú thích đăng nhập, favicon/logo dùng chung SVG và title riêng từng trang. Lint/typecheck/build/18 tests PASS; browser mở được Google qua Cognito, chưa kiểm chứng đăng nhập bằng tài khoản Google.
Review fix Task4.2 — complete (implementation/local): execution name/ARN được theo dõi trước StartExecution; cleanup chỉ xóa khi xác nhận terminal. Tám lifecycle regression tests và PS offline PASS; PR #77 đã merge main501c049; [PR fix #78](https://github.com/TrieuNguyenPhu/shorten-link/pull/78) từ main, CI #186 PASS, chờ review.

Phase 4 — complete (implementation/local). Task4.1/PR #76 đã merge mainf6e3769; Task4.2 có Terraform WebSocket/Step Functions, bounded IAM và E2E scenario, 37plan tests/Go checks/PS offline PASS; [PR #77](https://github.com/TrieuNguyenPhu/shorten-link/pull/77) CI #183 PASS, đã merge main501c049. AWS Phase3/4 chưa apply; Phase 5 đang triển khai, Task 5.1 và Task 5.3 đã hoàn tất kiểm tra local.

## Phases & Tasks

### Phase 0: Reference Audit, Clean Reset & Architecture Specification
- [x] **Task 0.1:** Hoàn thiện audit nguồn, xác minh setup skills/`AGENTS.md`, đặc tả kiến trúc và kế hoạch chi tiết sau review PR #65. AWS cũ đã được người dùng xác nhận dọn sạch; không lặp lại thao tác xóa.
- **Status:** complete (PR #65 đã merge main; chưa triển khai ứng dụng)

### Phase 1: Monorepo & Local Foundation
- **Task hiện tại:** Task 1.1/1.2 — complete và đã vào main; final review đã sửa/kiểm thử
- [x] **Task 1.1:** Khởi tạo cấu trúc Monorepo, `Makefile`, Go module (`backend/`), Domain Models, Base62 Generator (`crypto/rand`) & URL Validator kèm unit tests và CI nền.
- [x] **Task 1.2:** Xây dựng Observability Middleware (`slog` JSON, CloudWatch EMF metrics, X-Ray wrapper) & script đóng gói Lambda `bootstrap` ZIP (`arm64`) kèm tests.
- **Status:** complete (Task 1.1/1.2 đã vào main; final review đã sửa/kiểm thử)

### Phase 2: Core URL Shortener (DynamoDB, Cognito JWT & 4 HTTP Handlers)
- **Task hiện tại:** Task 2.2 và final review/deployment — complete (PR #73 chờ review/merge)
- [x] **Task 2.1:** Triển khai DynamoDB Repository, Link Service, EventBridge publisher, local HTTP harness & 4 Go HTTP Lambda Handlers kèm unit/integration tests (multi-tenant, anti-IDOR, collision retry, 301, lỗi publish).
- [x] **Task 2.2:** Xây dựng Terraform Bootstrap (S3 remote state + lockfile, OIDC provider/deployment role) & Core Infrastructure (`dynamodb`, Cognito Google-only social IdP, `lambda`, `http_api`, custom event bus) + `environments/dev`.
- **Status:** complete (implementation + AWS core apply/smoke; Google authenticated E2E pending)

### Phase 3: Event-Driven Analytics (EventBridge, SQS + DLQ, ClickAggregator)
- **Task hiện tại:** Task 3.2 — complete (PR #75 đã merge main6dee1ca; AWS pipeline chưa deploy)
- [x] **Task 3.1:** Triển khai Go Lambda `click-aggregator` (`TransactWriteItems` 3 thao tác, phân biệt chính xác `CancellationReasons`, `ReportBatchItemFailures`) kèm unit/integration tests.
- [x] **Task 3.2:** Xây dựng Terraform Module `event_pipeline` nhận custom bus từ Task 2.2, thêm rule `link.clicked`, SQS `ClickQueue` + `ClickDLQ` 14d, Event Source Mapping.
- **Status:** complete (implementation/local; AWS pipeline chưa deploy)

### Phase 4: Realtime WebSocket & Step Functions Moderation
- **Task hiện tại:** Task 4.2 — complete (implementation/local)
- [x] **Task 4.1:** Triển khai Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher (xử lý `410 GoneException`) & `moderate-check` + `statemachine/moderation.asl.json` kèm tests.
- [x] **Task 4.2:** Xây dựng Terraform Modules `websocket_api` (`$connect`, `$disconnect`, stage `prod`) & `moderation_sfn` (Step Functions Standard + IAM).
- **Status:** complete (implementation/local; AWS deployment/E2E pending)

### Phase 5: Frontend Next.js Dashboard & S3 + CloudFront OAC Hosting
- [x] **Task 5.1 — complete (implementation/local):** Hoàn thiện Next.js App Router (`output: "export"`), nút Google qua Cognito PKCE & các trang Landing (`/`), Login (`/login`), Callback (`/auth/callback`) theo Hand-Drawn / Doodle & Sketch UI.
- [x] **Task 5.2:** Xây dựng Dashboard (`/dashboard`), My Links (`/links`) cùng phong cách Doodle & Sketch, tích hợp HTTP API thật & Native WebSocket Hook (`useRealtimeClicks`); phần refresh Shrnk tiếp tục ở PR #82.
- [x] **Task 5.3 — complete (implementation/local):** Xây dựng Terraform Module `frontend_hosting` (Private S3 + CloudFront OAC + Cache Policies + CloudFront Function URL rewrite) trên nhánh PR mới từ `origin/main`.
- **Status:** complete (implementation/deployed; Google account E2E chưa thực hiện)

### Phase 6: Production Operations, Observability, Canary CI/CD & Load Testing
- [x] **Task 6.1:** Xây dựng Terraform Modules `observability` (CloudWatch Alarms, SNS, Dashboard, X-Ray) & `codedeploy_canary` (`LambdaCanary10Percent5Minutes` trên alias `live`).
- [x] **Task 6.2:** Thiết lập GitHub Actions CI/CD (AWS OIDC, Canary Deploy, S3/CloudFront Sync), kịch bản `tests/load/redirect.k6.js` & tài liệu vận hành E2E.
- **Status:** complete (implementation/local); AWS/CD/canary thật pending

## Key Decisions
| Decision | Rationale |
|----------|-----------|
| Tách domain CloudFront (Frontend) và API Gateway (`GET /{code}`) | Giữ nguyên route `GET /{code}` ở gốc API Gateway như bản gốc mà không xung đột với các route tĩnh của Next.js trên CloudFront. |
| Phân loại đầy đủ `CancellationReasons` theo META/STAT/marker | Không coi mọi cancellation là duplicate; phân biệt event của link đã xóa, conflict và lỗi hỗn hợp theo hợp đồng kiến trúc. |
| Cấu hình `lifecycle { ignore_changes = [function_version, routing_config] }` trên `aws_lambda_alias.live` | Cho phép CodeDeploy quản lý việc dịch chuyển phiên bản Canary mà không bị `terraform apply` ghi đè. |
| Giữ TTL marker 24 giờ và công bố giới hạn replay | Chống trùng phụ thuộc marker còn tồn tại; không khẳng định exactly-once vô thời hạn. |
| Bootstrap OIDC tách ứng dụng, build static frontend sau khi có outputs | Tài khoản sạch cần trust/role trước khi CI authenticate; static export cần endpoint thật ở thời điểm build. |
| Chỉ Google qua Cognito User Pool | Google redirect về Cognito `/oauth2/idpresponse`, Cognito Code+PKCE về CloudFront; API dùng Cognito access token + `sub`. Không có password/signup flow. Google client secret chỉ ở input deploy bảo mật và remote state. |
| Hand-Drawn / Doodle & Sketch UI | Dùng ngôn ngữ nét vẽ tay nhất quán cho frontend, nhưng giữ chữ, dữ liệu và các trạng thái tương tác rõ, accessible. |

## Errors Encountered (Nhật ký lịch sử trước phiên hoàn thiện 2026-10-27; không kiểm chứng AWS lại trong phiên này)
| Error | Attempt | Resolution |
|-------|---------|------------|
| `aws_dynamodb_table` / `aws_cognito_user_pool` `deletion_protection` chặn `terraform destroy` trên stack cũ | 1 | Tắt `deletion_protection` và bật `force_destroy = true` kèm `AWS_ACCOUNT_ID_ENDPOINT_MODE=disabled`, sau đó chạy `terraform destroy` xóa sạch 64 resources ở `dev` và 5 resources ở `bootstrap`. |
| Thư mục `services/lambda` và `.codegraph` bị khóa bởi tiến trình `go` / `codegraph` nền khi xóa local | 1 | Dừng tiến trình nền đang giữ khóa và xóa sạch toàn bộ file/thư mục cũ trong workspace. |


## Session 2026-11-05 — Recreate Phase 5 / Task 5.3 PR

- Không cherry-pick merge commit của PR #80. Branch mới được dựng trực tiếp từ `origin/main` (`702b97a`) rồi chép riêng 14 file implementation/test của hosting.
- Commit `20caa84` giữ phạm vi Terraform/S3/CloudFront OAC, không đưa thay đổi backend hoặc tài liệu conflict cũ vào branch.
- `terraform fmt -check -recursive`, `terraform validate` và `terraform test` PASS (19/19).
- Đã mở [PR #81](https://github.com/TrieuNguyenPhu/shorten-link/pull/81) từ branch mới; GitHub xác nhận không còn conflict với `main`.

## Task 5.4: Shrnk frontend refresh
- [x] Branding, auth-aware navigation, architecture route, landing create form và analytics-only dashboard.
- **Status:** implementation/local complete; PR [#82](https://github.com/TrieuNguyenPhu/shorten-link/pull/82) được đồng bộ với `main`.


## 2026-11-05 — UI refinement, project review và production rollout

- **Status: complete (implementation/deployed)**. Yêu cầu: đăng nhập về trang chủ; sửa dashboard nhấp nháy; layout chung; menu active/hover; login tối giản; font Lobster; review toàn project, sửa lỗi và triển khai production.
- Nhánh `fix/phase-05-ui-review-production` tạo từ `origin/main` (`2253a3c`), kế thừa hai commit sửa PKCE #87 để giữ luồng đăng nhập an toàn. Giữ nguyên các diff có sẵn để rà soát, không tự hoàn nguyên.

## 2026-11-05 — Đồng bộ PR #87
- **Status: complete (local)**. Sửa conflict bốn file trạng thái với main sau PR #86; giữ lịch sử hai bên và publisher/hosting checks đã merge.


## 2026-11-05 — Đồng bộ PR #88
- **Status: complete (local)**. Merge bản #87 đã đồng bộ main, bảo toàn UI/PKCE và toàn bộ lịch sử rollout.


## 2026-11-05 — PR #88 sau squash PR #87
- [x] **Status: complete (local)**. Merge main `2b6ed24` vào PR #88 hiện có; bảo toàn implementation UI/PKCE đã kiểm thử.

Task 6.2: [PR #90](https://github.com/TrieuNguyenPhu/shorten-link/pull/90), code tree `94d64f2` khớp local/remote. CI đang chạy; chưa merge/dispatch/apply. Merge #89 trước #90; giữ nguyên checkout/diff của người dùng.
