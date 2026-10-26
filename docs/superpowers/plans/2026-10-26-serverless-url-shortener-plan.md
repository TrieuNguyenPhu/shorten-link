# Serverless URL Shortener with Realtime Analytics — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` or `superpowers:executing-plans` together with `planning-with-files` and `ponytail` to implement this plan task-by-task. Every completed task MUST update `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, and `findings.md`, and be submitted as its own dedicated Pull Request.

**Goal:** Xây dựng hoàn chỉnh hệ thống Serverless URL Shortener with Realtime Analytics trên AWS bằng Golang (`provided.al2023`, `arm64`), Terraform (HCL), và Next.js Static Export (`S3 + CloudFront OAC`), bám sát kiến trúc và nghiệp vụ của chuỗi 21 bài và repository tham chiếu `nghiadaulau/serverless-url-shortener-aws`.

**Architecture:** Kiến trúc hướng sự kiện (Event-Driven Serverless) trên AWS gồm: giao diện tĩnh Next.js phân phối qua CloudFront + S3 (OAC), xác thực người dùng qua Cognito JWT, 4 route HTTP API Gateway và 2 route WebSocket API Gateway kết nối tới 8 hàm Lambda viết bằng Go dùng chung một bảng DynamoDB Single-Table (`PK`/`SK` + Sparse `GSI1` + `ttl`), xử lý đếm click bất đồng bộ qua EventBridge → SQS (+DLQ) → ClickAggregator (`TransactWriteItems` idempotent + đẩy WebSocket realtime), kèm workflow kiểm duyệt link bằng Step Functions và triển khai Canary qua CodeDeploy.

**Tech Stack:** Golang 1.24+, AWS SDK for Go v2, `aws-lambda-go`, Next.js App Router (`output: "export"`), TypeScript, Tailwind CSS, pnpm, Terraform 1.10+, GitHub Actions (AWS OIDC), k6.

**Spec:**
- `docs/phase-0-reference-audit.md`
- `docs/architecture-and-contracts.md`
- `AGENTS.md`

---

## Global Constraints

1. **Runtime & Binary:** Mọi Lambda viết bằng Go chạy trên `provided.al2023`, kiến trúc `arm64`, handler `bootstrap`, đóng gói thành file ZIP riêng cho từng hàm.
2. **Infrastructure as Code:** 100% tài nguyên AWS do Terraform quản lý; không dùng AWS SAM hoặc CloudFormation trực tiếp; tách remote state bootstrap khỏi application stack; region mặc định `ap-southeast-1` (có thể cấu hình qua biến).
3. **Database:** Một bảng DynamoDB duy nhất (`PAY_PER_REQUEST`, `PK`, `SK`, `GSI1` với `GSI1PK`/`GSI1SK`, TTL trên thuộc tính `ttl`). Không bao giờ dùng `Scan` cho các truy vấn đã có `GSI1`.
4. **Frontend:** Next.js App Router cấu hình `output: "export"`, triển khai trên S3 private bucket qua CloudFront OAC. Không dùng SSR, Server Actions, hoặc API Routes.
5. **Quy trình Git & PR:** Hoàn thành xong mỗi Task phải cập nhật trạng thái dự án (`PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`) và tạo một **Pull Request riêng** trên GitHub.
6. **An toàn chi phí AWS:** Không tự ý chạy `terraform apply` hoặc chạy load test lớn trên AWS khi chưa có yêu cầu xác nhận của người dùng.

## Review Focus

1. **Phân biệt lỗi `TransactionCanceledException` trong `click-aggregator`:** Chỉ coi là duplicate và bỏ qua khi `CancellationReasons[2].Code == "ConditionalCheckFailed"` (trùng `CLICK#<eventId>`); nếu hủy do `TransactionConflict` trên `LINK#<code>` hoặc `STAT#<date>`, phải trả lỗi để SQS retry record đó qua `ReportBatchItemFailures`.
2. **Cô lập lỗi WebSocket khỏi bộ đếm click:** Lỗi mạng khi gọi `PostToConnection` không được làm fail record SQS hoặc kích hoạt tăng bộ đếm lần hai; riêng lỗi HTTP `410 GoneException` phải tự động xóa item `CONN#<connectionId>` khỏi DynamoDB.
3. **Chống IDOR tuyệt đối khi xóa và liệt kê link:** `ownerId` chỉ lấy từ JWT claim `sub`; lệnh `DeleteItem` bắt buộc có `ConditionExpression: "attribute_exists(PK) AND ownerId = :me"` và trả cùng mã `404` cho cả trường hợp không tồn tại lẫn không thuộc sở hữu.
4. **Tách biệt Domain Redirect và Domain Frontend:** `shortUrl` trả về từ `POST /links` phải lấy từ `PUBLIC_REDIRECT_BASE_URL` của API Gateway, không lấy nhầm domain CloudFront của trang tĩnh.
5. **Phối hợp Terraform và CodeDeploy trên alias `live`:** Resource `aws_lambda_alias.live` của `resolve-link` phải cấu hình `lifecycle { ignore_changes = [function_version, routing_config] }` để `terraform apply` không ghi đè phiên bản đang được CodeDeploy dịch chuyển Canary.

---

## Chi Tiết Các Giai Đoạn & Tasks (Phase 0 → Phase 6)

### Phase 0 — Reference Audit, Clean Reset & Architecture Specification

#### Task 0.1: Dọn sạch tài nguyên cũ, cài đặt 13 bộ Skills, thiết lập `AGENTS.md`, Audit bản gốc & Lập kế hoạch tổng thể
- **Branch:** `chore/phase-00-reset-and-architecture-plan`
- **Files:**
  - Create: `.gitignore`
  - Create: `.agents/skills.json`
  - Create: `AGENTS.md`
  - Create: `docs/phase-0-reference-audit.md`
  - Create: `docs/architecture-and-contracts.md`
  - Create: `docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`
  - Create: `task_plan.md`
  - Create: `findings.md`
  - Create: `progress.md`
  - Create: `PROJECT_STATUS.md`
  - Create: `README.md`
- **Steps:**
  - [x] **Step 1:** Xóa toàn bộ tài nguyên AWS đã deploy trên stack cũ (`terraform destroy` cho `dev` và `bootstrap`, kiểm tra bằng AWS CLI).
  - [x] **Step 2:** Xóa toàn bộ file và thư mục cũ tại workspace local (giữ lại `.git` để quản lý branch/PR).
  - [x] **Step 3:** Cài đặt đầy đủ 13 bộ skills vào `~/.agents/skills` và cấu hình `~/.gemini/config/skills.json` + `.agents/skills.json`.
  - [x] **Step 4:** Soạn thảo `AGENTS.md`, báo cáo audit `docs/phase-0-reference-audit.md`, đặc tả kiến trúc `docs/architecture-and-contracts.md`, và kế hoạch chi tiết.
  - [x] **Step 5:** Khởi tạo `task_plan.md`, `findings.md`, `progress.md`, `PROJECT_STATUS.md`, `README.md`, commit, push và tạo Pull Request riêng cho Task 0.1.

---

### Phase 1 — Monorepo & Local Foundation

#### Task 1.1: Khởi tạo cấu trúc Monorepo, Go Domain Models, Base62 Generator & URL Validator
- **Branch:** `feat/phase-01-task-01-monorepo-and-domain`
- **Files:**
  - Create: `Makefile`
  - Create: `backend/go.mod`
  - Create: `backend/internal/domain/models.go`
  - Create: `backend/internal/domain/shortcode.go`
  - Create: `backend/internal/domain/shortcode_test.go`
  - Create: `backend/internal/config/config.go`
  - Create: `backend/internal/config/config_test.go`
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Interfaces:**
  - Produces:
    - `domain.Link`, `domain.DailyStat`, `domain.ClickEvent`, `domain.WSConnection`, `domain.ModerationCheck`
    - `domain.GenerateShortCode(length int) (string, error)` (7-char Base62 dùng `crypto/rand`)
    - `domain.NormalizeURL(raw string) (string, error)` (chỉ chấp nhận `http`/`https` hợp lệ)
    - `config.Load() (*config.Config, error)` (`TableName`, `EventBusName`, `WSEndpoint`, `PublicRedirectBaseURL`)
- **Steps:**
  - [ ] **Step 1:** Viết unit tests trong `backend/internal/domain/shortcode_test.go` và `backend/internal/config/config_test.go` kiểm tra sinh mã Base62 dài 7 ký tự, phân phối ký tự hợp lệ, và từ chối các URL không phải `http`/`https` (`javascript:`, `ftp:`, chuỗi rỗng, thiếu host).
  - [ ] **Step 2:** Chạy `go -C backend test ./...` để xác nhận test fail trước khi cài đặt.
  - [ ] **Step 3:** Triển khai `models.go`, `shortcode.go`, `config.go` và `Makefile` chuẩn cho toàn bộ monorepo.
  - [ ] **Step 4:** Chạy `go -C backend test -race -v ./...` và `go -C backend vet ./...` đảm bảo PASS 100%.
  - [ ] **Step 5:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 1.1.

#### Task 1.2: Observability Middleware (Structured JSON Logs, EMF Metrics, X-Ray) & Script Đóng Gói Lambda `arm64`
- **Branch:** `feat/phase-01-task-02-observability-and-packaging`
- **Files:**
  - Create: `backend/internal/middleware/logger.go`
  - Create: `backend/internal/middleware/emf.go`
  - Create: `backend/internal/middleware/xray.go`
  - Create: `backend/internal/middleware/middleware_test.go`
  - Create: `scripts/build-lambdas.ps1`
  - Create: `scripts/build-lambdas.sh`
  - Modify: `Makefile`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Interfaces:**
  - Produces:
    - `middleware.NewLogger(serviceName string) *slog.Logger` (kèm correlation/request ID)
    - `middleware.EmitCountMetric(w io.Writer, namespace, service, metricName string, value float64)` (chuẩn CloudWatch Embedded Metric Format JSON)
    - `middleware.InstrumentAWSConfig(cfg *aws.Config)`
- **Steps:**
  - [ ] **Step 1:** Viết unit tests trong `backend/internal/middleware/middleware_test.go` kiểm tra định dạng JSON của `slog` (có `service`, `requestId`, `level`, `msg`) và cấu trúc JSON `_aws.CloudWatchMetrics` của EMF (`Namespace: "UrlShortener"`, metrics `LinkResolved` / `LinkNotFound`).
  - [ ] **Step 2:** Triển khai `logger.go`, `emf.go`, `xray.go` và chạy `go -C backend test -race -v ./internal/middleware/...` đạt PASS.
  - [ ] **Step 3:** Viết script biên dịch chéo Go sang `GOOS=linux GOARCH=arm64 CGO_ENABLED=0` tạo file `bootstrap` và nén thành `<handler>.zip` có quyền thực thi chuẩn Linux (`0755`).
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và tạo Pull Request riêng cho Task 1.2.

---

### Phase 2 — Core URL Shortener (DynamoDB, Cognito JWT & 4 HTTP Handlers)

#### Task 2.1: DynamoDB Single-Table Repository, Link Service & 4 Go HTTP Lambda Handlers (`create-link`, `resolve-link`, `list-links`, `delete-link`)
- **Branch:** `feat/phase-02-task-01-core-crud-lambdas`
- **Files:**
  - Create: `backend/internal/repository/dynamodb.go`
  - Create: `backend/internal/repository/memory.go`
  - Create: `backend/internal/repository/repository_test.go`
  - Create: `backend/internal/service/link_service.go`
  - Create: `backend/internal/service/link_service_test.go`
  - Create: `backend/internal/httpapi/handler.go`
  - Create: `backend/internal/httpapi/handler_test.go`
  - Create: `backend/cmd/create-link/main.go`
  - Create: `backend/cmd/resolve-link/main.go`
  - Create: `backend/cmd/list-links/main.go`
  - Create: `backend/cmd/delete-link/main.go`
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Interfaces:**
  - Consumes: `domain.*`, `config.Config`, `middleware.*`
  - Produces:
    - `repository.LinkRepository` (`CreateLink`, `GetLink`, `ListUserLinks`, `DeleteUserLink`)
    - 4 binary entrypoints cho `POST /links`, `GET /{code}`, `GET /links`, `DELETE /links/{code}`
- **Steps:**
  - [ ] **Step 1:** Viết bộ test đầy đủ trong `link_service_test.go` và `handler_test.go` bao phủ:
    - Chưa có JWT `sub` → trả `401`
    - URL sai định dạng → trả `400`
    - Trùng mã ngắn → tự động retry tối đa 5 lần; quá 5 lần trả `500`
    - Tạo thành công → trả `201 {"code", "shortUrl", "target"}` với `shortUrl` dùng `PublicRedirectBaseURL`
    - User A chỉ xem được danh sách link của User A (mới nhất trước)
    - User B xóa link của User A → bị chặn bởi `ConditionExpression`, trả `404`
    - `GET /{code}` tồn tại → phát sự kiện `link.clicked` và trả `301` kèm header `Location`; không tồn tại → trả `404`
  - [ ] **Step 2:** Triển khai `dynamodb.go` (sử dụng đúng biểu thức `attribute_not_exists(PK)`, `GSI1PK = USER#<sub_id>`, `attribute_exists(PK) AND ownerId = :me`), `memory.go`, `link_service.go`, `httpapi/handler.go` và 4 `cmd/*/main.go`.
  - [ ] **Step 3:** Chạy `go -C backend test -race -v ./...` và thử build cả 4 gói ZIP `arm64`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 2.1.

#### Task 2.2: Terraform Bootstrap & Core Infrastructure Modules (`dynamodb`, `cognito`, `http_api`, `lambda`)
- **Branch:** `feat/phase-02-task-02-terraform-core-infra`
- **Files:**
  - Create: `infrastructure/terraform/bootstrap/versions.tf`
  - Create: `infrastructure/terraform/bootstrap/variables.tf`
  - Create: `infrastructure/terraform/bootstrap/main.tf`
  - Create: `infrastructure/terraform/bootstrap/outputs.tf`
  - Create: `infrastructure/terraform/modules/dynamodb/main.tf`
  - Create: `infrastructure/terraform/modules/cognito/main.tf`
  - Create: `infrastructure/terraform/modules/lambda/main.tf`
  - Create: `infrastructure/terraform/modules/http_api/main.tf`
  - Create: `infrastructure/terraform/environments/dev/versions.tf`
  - Create: `infrastructure/terraform/environments/dev/variables.tf`
  - Create: `infrastructure/terraform/environments/dev/main.tf`
  - Create: `infrastructure/terraform/environments/dev/outputs.tf`
  - Create: `infrastructure/terraform/environments/dev/terraform.tfvars.example`
  - Create: `infrastructure/terraform/environments/dev/backend.hcl.example`
  - Modify: `Makefile`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Thiết kế module `bootstrap` tạo S3 bucket lưu remote state với versioning, mã hóa AES256, chặn public access và hỗ trợ `use_lockfile = true`.
  - [ ] **Step 2:** Thiết kế các module `dynamodb` (`PK`/`SK`, `GSI1`, `ttl`), `cognito` (User Pool, Domain, App Client hỗ trợ OAuth2 Code PKCE + User Auth), `lambda` (`provided.al2023`, `arm64`, IAM least-privilege riêng cho từng hàm), và `http_api` (4 routes, Cognito JWT Authorizer, CORS, Throttling).
  - [ ] **Step 3:** Chạy `terraform fmt -check` và `terraform validate` trên cả `bootstrap` và `environments/dev`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 2.2.

---

### Phase 3 — Event-Driven Analytics (EventBridge, SQS + DLQ, ClickAggregator)

#### Task 3.1: Go Lambda `click-aggregator` (Transactional Idempotency, `CancellationReasons` Inspection & Partial Batch Failure)
- **Branch:** `feat/phase-03-task-01-click-aggregator-lambda`
- **Files:**
  - Create: `backend/internal/events/eventbridge.go`
  - Create: `backend/internal/service/aggregator_service.go`
  - Create: `backend/internal/service/aggregator_service_test.go`
  - Create: `backend/cmd/click-aggregator/main.go`
  - Modify: `backend/internal/repository/dynamodb.go`, `backend/internal/repository/memory.go`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Interfaces:**
  - Consumes: `events.SQSEvent` (chứa EventBridge envelope `source: "urlshortener"`, `detail-type: "link.clicked"`)
  - Produces: `events.SQSEventResponse` (`BatchItemFailures`)
- **Steps:**
  - [ ] **Step 1:** Viết unit/integration tests trong `aggregator_service_test.go` kiểm chứng:
    - Sự kiện hợp lệ tăng `META.clicks` lên 1, tăng `STAT#YYYY-MM-DD.count` lên 1, và tạo `CLICK#<eventId>` với `ttl = now + 86400`.
    - Sự kiện trùng `eventId` (`CancellationReasons[2].Code == "ConditionalCheckFailed"`) được bỏ qua an toàn, không tăng click lần hai và **không** nằm trong `BatchItemFailures`.
    - Lỗi `TransactionCanceledException` do `TransactionConflict` hoặc lỗi mạng DynamoDB **bắt buộc** đưa `messageId` vào `BatchItemFailures` để SQS retry.
    - Trong một batch 10 messages có 2 messages lỗi và 8 messages hợp lệ, chỉ đúng 2 `messageId` lỗi được trả về trong `BatchItemFailures`.
    - Khi notifier WebSocket trả lỗi, bộ đếm đã ghi không bị ảnh hưởng và record SQS vẫn thành công.
  - [ ] **Step 2:** Triển khai `RecordClickTransaction` trong `dynamodb.go`, `aggregator_service.go` và `backend/cmd/click-aggregator/main.go`.
  - [ ] **Step 3:** Chạy `go -C backend test -race -v ./...` và đóng gói `click-aggregator.zip`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 3.1.

#### Task 3.2: Terraform Module `event_pipeline` (EventBridge Custom Bus, Rule, SQS `ClickQueue` + `ClickDLQ`, Event Source Mapping)
- **Branch:** `feat/phase-03-task-02-terraform-event-pipeline`
- **Files:**
  - Create: `infrastructure/terraform/modules/event_pipeline/main.tf`
  - Create: `infrastructure/terraform/modules/event_pipeline/variables.tf`
  - Create: `infrastructure/terraform/modules/event_pipeline/outputs.tf`
  - Modify: `infrastructure/terraform/environments/dev/main.tf`, `infrastructure/terraform/environments/dev/outputs.tf`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Tạo module `event_pipeline` gồm custom bus `url-shortener-events`, rule khớp `source=["urlshortener"]` & `detail-type=["link.clicked"]`, SQS `ClickQueue` (`visibility_timeout_seconds = 30`, `maxReceiveCount = 3`), `ClickDLQ` (`message_retention_seconds = 1209600` — 14 ngày), `aws_sqs_queue_policy` giới hạn theo `aws:SourceArn`, và `aws_lambda_event_source_mapping` (`batch_size = 10`, `function_response_types = ["ReportBatchItemFailures"]`).
  - [ ] **Step 2:** Kết nối `click-aggregator` Lambda và quyền IAM least-privilege vào `environments/dev`.
  - [ ] **Step 3:** Chạy `terraform fmt -check` và `terraform validate`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 3.2.

---

### Phase 4 — Realtime WebSocket & Step Functions Moderation

#### Task 4.1: Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher & `moderate-check` + ASL State Machine
- **Branch:** `feat/phase-04-task-01-websocket-and-moderation-lambdas`
- **Files:**
  - Create: `backend/internal/events/ws_pusher.go`
  - Create: `backend/internal/events/ws_pusher_test.go`
  - Create: `backend/internal/service/ws_service.go`
  - Create: `backend/internal/service/ws_service_test.go`
  - Create: `backend/internal/service/moderation_service.go`
  - Create: `backend/internal/service/moderation_service_test.go`
  - Create: `backend/cmd/ws-connect/main.go`
  - Create: `backend/cmd/ws-disconnect/main.go`
  - Create: `backend/cmd/moderate-check/main.go`
  - Create: `statemachine/moderation.asl.json`
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Viết unit tests cho:
    - `$connect` thiếu `?code=` → trả `400`; có `code` → lưu `PK=CONN#<id>, SK=META, code, GSI1PK=WSCODE#<code>, GSI1SK=CONN#<id>, ttl=now+7200` và trả `200`.
    - `$disconnect` → xóa `PK=CONN#<id>, SK=META` và trả `200`.
    - `PushClickUpdate(ctx, code)` → lấy tổng `clicks`, query `GSI1` theo `WSCODE#<code>`, gửi JSON `{"code": code, "clicks": clicks}` tới từng connection; nếu gặp `GoneException` (HTTP 410) thì tự động gọi `DeleteItem` dọn connection chết.
    - `moderate-check` → kiểm tra danh sách từ khóa cấm (`malware`, `phishing`, `evil`), trả `{code, target, safe}`.
  - [ ] **Step 2:** Triển khai code Go và file `statemachine/moderation.asl.json` giữ nguyên cấu trúc Retry, Catch và `dynamodb:updateItem` của bản gốc.
  - [ ] **Step 3:** Chạy `go -C backend test -race -v ./...` và build toàn bộ 8 Lambda binaries.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 4.1.

#### Task 4.2: Terraform Modules `websocket_api` & `moderation_sfn`
- **Branch:** `feat/phase-04-task-02-terraform-websocket-and-sfn`
- **Files:**
  - Create: `infrastructure/terraform/modules/websocket_api/main.tf`
  - Create: `infrastructure/terraform/modules/websocket_api/variables.tf`
  - Create: `infrastructure/terraform/modules/websocket_api/outputs.tf`
  - Create: `infrastructure/terraform/modules/moderation_sfn/main.tf`
  - Create: `infrastructure/terraform/modules/moderation_sfn/variables.tf`
  - Create: `infrastructure/terraform/modules/moderation_sfn/outputs.tf`
  - Modify: `infrastructure/terraform/environments/dev/main.tf`, `infrastructure/terraform/environments/dev/outputs.tf`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Tạo module `websocket_api` (`ProtocolType = WEBSOCKET`, route `$connect` và `$disconnect`, stage `prod` với `auto_deploy = true`, Lambda permissions) và cấp biến môi trường `WS_ENDPOINT` + quyền `execute-api:ManageConnections` cho `click-aggregator`.
  - [ ] **Step 2:** Tạo module `moderation_sfn` triển khai `aws_sfn_state_machine` (`type = "STANDARD"`) từ `statemachine/moderation.asl.json` kèm IAM role chỉ cấp quyền `lambda:InvokeFunction` trên `moderate-check` và `dynamodb:UpdateItem` trên bảng DynamoDB.
  - [ ] **Step 3:** Chạy `terraform fmt -check` và `terraform validate`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 4.2.

---

### Phase 5 — Frontend Next.js Dashboard & S3 + CloudFront OAC Hosting

#### Task 5.1: Khởi tạo Next.js App Router (`output: "export"`), Cognito PKCE Auth & Các Trang Giao Diện
- **Branch:** `feat/phase-05-task-01-nextjs-auth-and-pages`
- **Files:**
  - Create: `frontend/package.json`, `frontend/next.config.ts`, `frontend/tsconfig.json`, `frontend/postcss.config.mjs`
  - Create: `frontend/src/types/link.ts`
  - Create: `frontend/src/lib/config.ts`
  - Create: `frontend/src/lib/auth.ts`
  - Create: `frontend/src/components/Navbar.tsx`
  - Create: `frontend/src/app/layout.tsx`, `frontend/src/app/globals.css`
  - Create: `frontend/src/app/page.tsx` (Landing page)
  - Create: `frontend/src/app/login/page.tsx` (Login / Signup với Cognito Hosted UI PKCE & Direct Auth fallback)
  - Create: `frontend/src/app/auth/callback/page.tsx` (OAuth2 PKCE Callback)
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Khởi tạo project Next.js trong `frontend/` với `output: "export"`, `trailingSlash: true`, `images: { unoptimized: true }`.
  - [ ] **Step 2:** Triển khai module xác thực Cognito (`auth.ts`) hỗ trợ đầy đủ Authorization Code + PKCE qua Hosted UI (sinh `code_verifier`, `code_challenge` S256, đổi `code` lấy `id_token`/`access_token` tại `/oauth2/token`) cũng như đăng nhập trực tiếp, lưu trữ phiên an toàn và đăng xuất.
  - [ ] **Step 3:** Xây dựng Landing page, Login/Signup page và Auth Callback page với giao diện sạch, hiện đại, responsive.
  - [ ] **Step 4:** Chạy `pnpm --dir frontend lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend build` kiểm chứng xuất tĩnh thành công.
  - [ ] **Step 5:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 5.1.

#### Task 5.2: Dashboard, My Links, Tích Hợp HTTP API Thật & Native WebSocket Realtime Hook
- **Branch:** `feat/phase-05-task-02-dashboard-api-and-websocket`
- **Files:**
  - Create: `frontend/src/lib/api.ts`
  - Create: `frontend/src/hooks/useRealtimeClicks.ts`
  - Create: `frontend/src/features/links/CreateLinkForm.tsx`
  - Create: `frontend/src/features/links/LinksTable.tsx`
  - Create: `frontend/src/app/dashboard/page.tsx`
  - Create: `frontend/src/app/links/page.tsx`
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Xây dựng `api.ts` gọi trực tiếp `POST /links`, `GET /links`, `DELETE /links/{code}` kèm header `Authorization: Bearer <token>` và xử lý rõ ràng các trạng thái lỗi `400`, `401`, `404`, `429`, `500`.
  - [ ] **Step 2:** Xây dựng hook `useRealtimeClicks.ts` sử dụng `WebSocket` gốc của trình duyệt kết nối tới `${WS_URL}?code=${code}`, tự động reconnect theo exponential backoff khi rớt mạng, dọn dẹp kết nối khi link bị xóa hoặc component unmount, và kích hoạt hiệu ứng highlight khi nhận `{ code, clicks }`.
  - [ ] **Step 3:** Hoàn thiện trang `/dashboard` và `/links` (tạo link, copy short link, nút mở link/click thử, xóa link, tổng số clicks nhảy realtime, trạng thái kết nối WebSocket).
  - [ ] **Step 4:** Chạy `pnpm --dir frontend lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend build` và kiểm tra thư mục `frontend/out/`.
  - [ ] **Step 5:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 5.2.

#### Task 5.3: Terraform Module `frontend_hosting` (Private S3 + CloudFront OAC + Cache Policies + Static Rewrite)
- **Branch:** `feat/phase-05-task-03-terraform-s3-cloudfront-oac`
- **Files:**
  - Create: `infrastructure/terraform/modules/frontend_hosting/main.tf`
  - Create: `infrastructure/terraform/modules/frontend_hosting/variables.tf`
  - Create: `infrastructure/terraform/modules/frontend_hosting/outputs.tf`
  - Create: `infrastructure/terraform/modules/frontend_hosting/url-rewrite.js`
  - Modify: `infrastructure/terraform/environments/dev/main.tf`, `infrastructure/terraform/environments/dev/outputs.tf`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Tạo module `frontend_hosting` gồm S3 bucket private (`aws_s3_bucket_public_access_block` bật cả 4 cờ), `aws_cloudfront_origin_access_control` (SigV4 `always`), bucket policy chỉ cho phép CloudFront distribution ARN đọc object qua HTTPS.
  - [ ] **Step 2:** Cấu hình CloudFront cache behavior tách biệt: `_next/static/*` cache dài hạn (immutable 1 năm), còn HTML/default behavior có TTL ngắn để nhận bản cập nhật nhanh; kèm CloudFront Function `viewer-request` chuẩn hóa đường dẫn `/dashboard` → `/dashboard/index.html` mà **không** dùng custom error response catch-all `200`.
  - [ ] **Step 3:** Cập nhật CORS của HTTP API và callback URLs của Cognito tự động nhận domain CloudFront.
  - [ ] **Step 4:** Chạy `terraform fmt -check` và `terraform validate`.
  - [ ] **Step 5:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 5.3.

---

### Phase 6 — Production Operations, Observability, Canary CI/CD & Load Testing

#### Task 6.1: Terraform Modules `observability` (CloudWatch Alarms, SNS, Dashboard, X-Ray) & `codedeploy_canary`
- **Branch:** `feat/phase-06-task-01-observability-and-canary`
- **Files:**
  - Create: `infrastructure/terraform/modules/observability/main.tf`
  - Create: `infrastructure/terraform/modules/observability/variables.tf`
  - Create: `infrastructure/terraform/modules/observability/outputs.tf`
  - Create: `infrastructure/terraform/modules/codedeploy_canary/main.tf`
  - Create: `infrastructure/terraform/modules/codedeploy_canary/variables.tf`
  - Create: `infrastructure/terraform/modules/codedeploy_canary/outputs.tf`
  - Modify: `infrastructure/terraform/modules/lambda/main.tf`, `infrastructure/terraform/environments/dev/main.tf`, `infrastructure/terraform/environments/dev/outputs.tf`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Tạo module `observability` gồm SNS `AlarmTopic`, `ResolveThrottleAlarm`, `ResolveErrorAlarm`, `ClickDLQVisibleMessagesAlarm`, và `aws_cloudwatch_dashboard` hiển thị Invocations/Errors/Throttles, Duration p50/p99, SQS backlog/DLQ và custom metric `UrlShortener/LinkResolved`.
  - [ ] **Step 2:** Tạo module `codedeploy_canary` (`aws_codedeploy_app` compute platform `Lambda`, `aws_codedeploy_deployment_group` với `LambdaCanary10Percent5Minutes`, tự động rollback khi `ResolveErrorAlarm` hoặc `ResolveThrottleAlarm` kích hoạt) và cấu hình `aws_lambda_alias.live` với `lifecycle { ignore_changes = [function_version, routing_config] }`.
  - [ ] **Step 3:** Chạy `terraform fmt -check` và `terraform validate`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 6.1.

#### Task 6.2: GitHub Actions CI/CD (AWS OIDC, Canary Deploy, S3/CloudFront Sync), k6 Load Test & Tài Liệu Vận Hành E2E
- **Branch:** `feat/phase-06-task-02-cicd-k6-and-runbook`
- **Files:**
  - Create: `.github/workflows/ci.yml`
  - Create: `.github/workflows/deploy.yml`
  - Create: `scripts/deploy-canary.sh`
  - Create: `tests/load/redirect.k6.js`
  - Create: `docs/e2e-verification-and-operations.md`
  - Modify: `README.md`, `Makefile`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Xây dựng `.github/workflows/ci.yml` tự động chạy trên mỗi PR/push: Go test (`-race`), Go vet, build 8 gói ZIP `arm64`, Next.js lint/typecheck/static build, và `terraform fmt -check` + `terraform validate`.
  - [ ] **Step 2:** Xây dựng `.github/workflows/deploy.yml` xác thực AWS qua OIDC (`aws-actions/configure-aws-credentials@v4`), chạy `terraform apply`, kích hoạt CodeDeploy canary deployment cho `resolve-link` alias `live` kèm theo dõi rollback, đồng bộ `frontend/out/` lên S3 và tạo CloudFront invalidation.
  - [ ] **Step 3:** Viết kịch bản `tests/load/redirect.k6.js` với chế độ mặc định tải nhỏ (`MODE=smoke`, 2 VUs trong 10s) và chế độ stress test (`MODE=stress`, ramp lên 40 VUs như Bài 20).
  - [ ] **Step 4:** Biên soạn hướng dẫn kiểm thử E2E, phân tích chi phí và quy trình dọn dẹp tài nguyên trong `docs/e2e-verification-and-operations.md`.
  - [ ] **Step 5:** Chạy toàn bộ lệnh kiểm tra (`make check`), cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 6.2.
