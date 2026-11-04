# Serverless URL Shortener with Realtime Analytics — Implementation Plan

> **Dành cho agent:** Dùng `superpowers:subagent-driven-development` hoặc `superpowers:executing-plans` cùng `planning-with-files` và `ponytail` theo từng task. Mỗi task hoàn tất phải cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md` và có PR riêng. Sửa review của cùng task tiếp tục branch/PR đang mở.

**Goal:** Xây dựng hoàn chỉnh hệ thống Serverless URL Shortener with Realtime Analytics trên AWS bằng Golang (`provided.al2023`, `arm64`), Terraform (HCL), và Next.js Static Export (`S3 + CloudFront OAC`), bám sát kiến trúc và nghiệp vụ của chuỗi 21 bài và repository tham chiếu `nghiadaulau/serverless-url-shortener-aws`.

**Architecture:** Kiến trúc hướng sự kiện (Event-Driven Serverless) trên AWS gồm: giao diện tĩnh Next.js phong cách Hand-Drawn / Doodle & Sketch UI phân phối qua CloudFront + S3 (OAC), đăng nhập Google duy nhất qua Cognito User Pool Code + PKCE/JWT, 4 route HTTP API Gateway và 2 route WebSocket API Gateway kết nối tới 8 hàm Lambda viết bằng Go dùng chung một bảng DynamoDB Single-Table (`PK`/`SK` + Sparse `GSI1` + `ttl`), xử lý đếm click bất đồng bộ qua EventBridge → SQS (+DLQ) → ClickAggregator (`TransactWriteItems` idempotent + đẩy WebSocket realtime), kèm workflow kiểm duyệt link bằng Step Functions và triển khai Canary qua CodeDeploy.

**Tech Stack:** Golang, AWS SDK for Go v2, `aws-lambda-go`, Next.js App Router (`output: "export"`), TypeScript, Tailwind CSS, pnpm, Terraform có S3 native lockfile, GitHub Actions (AWS OIDC), k6. Chọn phiên bản được hỗ trợ và pin toolchain/dependencies ở Task 1.1, 2.2 và 5.1; Phase 0 không xác nhận phiên bản mới nhất.

**Mốc:** đọc nguồn 2026-10-26, hoàn thiện đặc tả 2026-10-27. Phase 0 hoàn tất audit/kế hoạch, chưa chạy app hoặc kiểm thử AWS. Tất cả đường dẫn và lệnh kiểm thử các phase sau là đầu ra phải tạo, không phải bằng chứng đã chạy.

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

1. **Publish và transaction:** áp dụng đầy đủ [hợp đồng lỗi](../../architecture-and-contracts.md) mục 3.4 và 5.2: kiểm từng PutEvents entry, bounded retry; duplicate phải kiểm đủ ba cancellation reasons; late-deleted cần consistent read; mixed/unknown errors retry. Không rút gọn thành chỉ kiểm marker index2.
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
  - [x] **Step 1:** Ghi nhận người dùng xác nhận ngày 2026-10-26 đã xóa AWS cũ; phiên hoàn thiện Phase 0 không lặp destroy, kiểm kê AWS hoặc xác nhận hóa đơn.
  - [x] **Step 2:** Xóa toàn bộ file và thư mục cũ tại workspace local (giữ lại `.git` để quản lý branch/PR).
  - [x] **Step 3:** Đối chiếu cấu hình và file đại diện 13 bộ skills: 314 thư mục cấp đầu/376 SKILL.md đệ quy có trùng; không cài lại hay khẳng định mọi host tự nạp.
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
  - Create: `.github/workflows/ci.yml` (CI nền Go, mở rộng cùng từng phase)
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Interfaces:**
  - Produces:
    - `domain.Link`, `domain.DailyStat`, `domain.ClickEvent`, `domain.WSConnection`, `domain.ModerationCheck`
    - `domain.GenerateShortCode(length int) (string, error)` (7-char Base62 dùng `crypto/rand`)
    - `domain.NormalizeURL(raw string) (string, error)` (chỉ chấp nhận `http`/`https` hợp lệ)
    - `config.Load() (*config.Config, error)` (`TableName`, `EventBusName`, `WSEndpoint`, `PublicRedirectBaseURL`)
- **Steps:**
  - [x] **Step 1:** Unit tests kiểm tra Base62 7 ký tự, alphabet hợp lệ và URL http/https (từ chối javascript/ftp/rỗng/thiếu host); timestamp createdAt UTC cố định milliseconds và ordering cùng giây theo đặc tả4.2. Không dùng RFC3339Nano biến độ dài cho GSI sort key.
  - [x] **Step 2:** Chạy `go -C backend test ./...` để xác nhận test fail trước khi cài đặt.
  - [x] **Step 3:** Triển khai `models.go`, `shortcode.go`, `config.go`, `Makefile` và CI nền Go test/vet trên PR; pin toolchain/dependencies được hỗ trợ. Chỉ thêm check của thành phần đã tồn tại, mở rộng CI cùng mỗi phase.
  - [x] **Step 4:** Chạy `go -C backend test -race -v ./...` và `go -C backend vet ./...` đảm bảo PASS 100%.
  - [x] **Step 5:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 1.1.

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
  - [x] **Step 1:** Viết unit tests trong `backend/internal/middleware/middleware_test.go` kiểm tra định dạng JSON của `slog` (có `service`, `requestId`, `level`, `msg`) và cấu trúc JSON `_aws.CloudWatchMetrics` của EMF (`Namespace: "UrlShortener"`, metrics `LinkResolved` / `LinkNotFound`).
  - [x] **Step 2:** Xác minh support/compatibility instrumentation Go rồi triển khai `logger.go`, `emf.go`, `xray.go` và test middleware. [X-Ray Go SDK](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-go.html) đã vào maintenance từ 25/02/2026; đối chiếu [timeline chính thức](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-daemon-timeline.html) khi chọn instrumentation có export X-Ray. Không giả định tương đương đầy đủ Powertools Node hay mở rộng hạ tầng OTel ngoài phạm vi.
  - [x] **Step 3:** Viết script biên dịch chéo Go sang `GOOS=linux GOARCH=arm64 CGO_ENABLED=0` tạo file `bootstrap` và nén thành `<handler>.zip` có quyền thực thi chuẩn Linux (`0755`).
  - [x] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và tạo Pull Request riêng cho Task 1.2.

---

### Phase 2 — Core URL Shortener (DynamoDB, Cognito JWT & 4 HTTP Handlers)

#### Task 2.1: DynamoDB Single-Table Repository, Link Service & 4 Go HTTP Lambda Handlers (`create-link`, `resolve-link`, `list-links`, `delete-link`)
- **Branch:** `feat/phase-02-task-01-core-crud-lambdas`
- **Files:**
  - Create: `backend/internal/repository/dynamodb.go`
  - Create: `backend/internal/repository/repository_test.go`
  - Create: `backend/internal/repository/dynamodb_integration_test.go`
  - Create: `backend/internal/events/eventbridge.go`
  - Create: `backend/internal/events/eventbridge_test.go`
  - Create: `backend/tools/local-http/main.go` (adapter trong Go module, được import backend/internal và chạy handler thật)
  - Create: `scripts/local-http/config.example.json` (DynamoDB Local và endpoint test, không secret)
  - Create: `docs/local-testing.md`
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
    - URL sai, body null, thiếu url hoặc url không phải string → trả `400`
    - Trùng mã ngắn → tối đa 5 lần ghi tổng cộng gồm lần đầu; hết lượt trả `500`
    - Tạo thành công → trả `201 {"code", "shortUrl", "target"}` với `shortUrl` dùng `PublicRedirectBaseURL`
    - User A chỉ xem được danh sách link của User A (mới nhất trước)
    - User B xóa link của User A → bị chặn bởi `ConditionExpression`, trả `404`
    - Resolve thành công chỉ 301 sau EventId hợp lệ; missing link404; publish HTTP200/FailedEntryCount1, nonretryable, retry success/exhausted, entry thiếu/sai theo hợp đồng 3.4
    - List paginate đến hết LastEvaluatedKey; giữ JSON links, thứ tự và owner scope
  - [ ] **Step 2:** Triển khai `dynamodb.go`, `link_service.go`, `httpapi/handler.go`, EventBridge publisher và 4 `cmd/*/main.go` theo đặc tả; tạo local HTTP harness dùng handler/repository thật và DynamoDB Local, không production memory storage. Unit mock chỉ tại ranh giới SDK/publisher; harness không chứng minh JWT authorizer/IAM AWS.
  - [ ] **Step 3:** Chạy Go race/vet, integration DynamoDB Local và build 4 ZIP arm64; ghi lệnh khởi động harness, cấu hình test và bằng chứng vào `docs/local-testing.md`, mở rộng CI tương ứng.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 2.1.

#### Task 2.2: Terraform Bootstrap & Core Infrastructure Modules (`dynamodb`, `cognito`, `http_api`, `lambda`)
- **Branch:** `feat/phase-02-task-02-terraform-core-infra`
- **Files:**
  - Create: `infrastructure/terraform/bootstrap/versions.tf`
  - Create: `infrastructure/terraform/bootstrap/variables.tf`
  - Create: `infrastructure/terraform/bootstrap/main.tf`
  - Create: `infrastructure/terraform/bootstrap/outputs.tf`
  - Create: `infrastructure/terraform/bootstrap/oidc.tf`
  - Create: `infrastructure/terraform/bootstrap/tests/trust.tftest.hcl`
  - Create: `infrastructure/terraform/modules/event_bus/main.tf`
  - Create: `infrastructure/terraform/modules/event_bus/outputs.tf`
  - Create: `infrastructure/terraform/environments/dev/tests/core.tftest.hcl`
  - Create: `tests/e2e/aws-flow.ps1` (mở rộng scenario theo từng phase có hạ tầng)
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
  - [ ] **Step 1:** Bootstrap S3 state/versioning/AES256/public block/use_lockfile và GitHub OIDC provider/deployment role theo kiến trúc mục 6. Trust aud/sub chính xác repo/main hoặc protected environment; provider có sẵn import. Operator credentials/SSO bootstrap lần đầu khi được phép, state bootstrap độc lập; app role không sửa trust/bootstrap. Scope state object, lockfile, ListBucket và PassRole; output role ARN cho CD.
  - [ ] **Step 2:** Tạo modules `dynamodb`, `cognito` User Pool + domain + Google social IdP + public app client Code/PKCE, `lambda` provided.al2023/arm64, `http_api` 4 routes/JWT/CORS/throttling và custom `event_bus` url-shortener-events. Trước IdP apply, tạo Google OAuth Web client ngoài AWS với redirect URI Cognito `/oauth2/idpresponse`; Google client ID/secret cấp bằng secret input, không commit và bảo vệ remote state. App client chỉ `Google`, cho phép OAuth scopes `openid email profile`, không bật `explicit_auth_flows` dạng password; ba route riêng tư yêu cầu scope `openid` và Cognito access token. Truyền EVENT_BUS, PUBLIC_REDIRECT_BASE_URL, bus ARN/IAM PutEvents cho resolve ngay Phase 2. Pin provider/toolchain; tags thống nhất, region mặc định ap-southeast-1, memory128/timeout5, IAM từng hàm và Logs/X-Ray. Publish version/alias live ban đầu, API integration + invoke permission dùng qualified alias ARN. Output API/Cognito/bus/role cho phase sau; analytics chỉ end-to-end sau rule Phase 3.
  - [ ] **Step 3:** Chạy Terraform fmt/init-backend=false/validate/tests theo cổng kiểm tra cuối kế hoạch cho bootstrap/dev; CI mở rộng không apply AWS.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 2.2.

---

### Phase 3 — Event-Driven Analytics (EventBridge, SQS + DLQ, ClickAggregator)

#### Task 3.1: Go Lambda `click-aggregator` (Transactional Idempotency, `CancellationReasons` Inspection & Partial Batch Failure)
- **Branch:** `feat/phase-03-task-01-click-aggregator-lambda`
- **Files:**
  - Create: `backend/internal/service/aggregator_service.go`
  - Create: `backend/internal/service/aggregator_service_test.go`
  - Create: `backend/cmd/click-aggregator/main.go`
  - Modify: `backend/internal/repository/dynamodb.go`, `backend/internal/repository/dynamodb_integration_test.go`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Interfaces:**
  - Consumes: `events.SQSEvent` (chứa EventBridge envelope `source: "urlshortener"`, `detail-type: "link.clicked"`)
  - Produces: `events.SQSEventResponse` (`BatchItemFailures`)
- **Steps:**
  - [ ] **Step 1:** Viết unit/integration tests trong `aggregator_service_test.go` kiểm chứng:
    - Sự kiện hợp lệ tăng `META.clicks` lên 1, tăng `STAT#YYYY-MM-DD.count` lên 1, và tạo `CLICK#<eventId>` với `ttl = now + 86400`.
    - Duplicate đúng cả ba reasons, mixed reasons, empty/nil/unknown reasons, conflicts và throttle theo bảng phân loại kiến trúc5.2; không chỉ xét marker index2.
    - Delete race không tái tạo META/STAT/marker; chỉ ack late-deleted sau consistent read xác nhận, Get lỗi hoặc META còn thì retry. Marker TTL đã qua nhưng còn item vẫn dedupe; marker bị xóa rồi replay đếm lại.
    - Lỗi `TransactionCanceledException` do `TransactionConflict` hoặc lỗi mạng DynamoDB **bắt buộc** đưa `messageId` vào `BatchItemFailures` để SQS retry.
    - Trong một batch 10 messages có 2 messages lỗi và 8 messages hợp lệ, chỉ đúng 2 `messageId` lỗi được trả về trong `BatchItemFailures`.
    - Khi notifier WebSocket trả lỗi, bộ đếm đã ghi không bị ảnh hưởng và record SQS vẫn thành công.
  - [ ] **Step 2:** Triển khai `RecordClickTransaction` trong `dynamodb.go`, `aggregator_service.go` và `backend/cmd/click-aggregator/main.go`.
  - [ ] **Step 3:** Chạy Go race/vet và integration DynamoDB Local cho transaction, cập nhật CI và đóng gói `click-aggregator.zip`.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 3.1.

#### Task 3.2: Terraform Module `event_pipeline` (Nhận EventBridge Bus, Rule, SQS `ClickQueue` + `ClickDLQ`, Event Source Mapping)
- **Branch:** `feat/phase-03-task-02-terraform-event-pipeline`
- **Files:**
  - Create: `infrastructure/terraform/modules/event_pipeline/main.tf`
  - Create: `infrastructure/terraform/modules/event_pipeline/variables.tf`
  - Create: `infrastructure/terraform/modules/event_pipeline/outputs.tf`
  - Modify: `infrastructure/terraform/environments/dev/main.tf`, `infrastructure/terraform/environments/dev/outputs.tf`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Module nhận bus name/ARN từ Task 2.2, không tạo bus duplicate; thêm rule source=urlshortener/detail-type=link.clicked, ClickQueue visibility30/maxReceive3, ClickDLQ retention14d, queue policy SourceArn và ESM batch10/ReportBatchItemFailures. Giữ maxReceive3 có chủ đích dù AWS khuyến nghị >=5.
  - [ ] **Step 2:** Kết nối aggregator, khai báo quyền poll SQS ReceiveMessage/DeleteMessage/GetQueueAttributes scoped queue và quyền DynamoDB transaction; kiểm thử ESM/DLQ thực tế chỉ trong AWS được phép. Thêm assertions Terraform cho bus input, queue policy/redrive/ESM, mở rộng CI.
  - [ ] **Step 3:** Chạy cổng Terraform fmt/init-backend=false/validate/test ở cuối kế hoạch, cập nhật CI.
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
    - Pusher đọc META ConsistentRead, paginate GSI WSCODE mọi trang, gửi `{code, clicks}`; 410 cleanup, non410/Get/Query/cleanup lỗi đều được log và cô lập sau commit; public connect không có owner auth.
    - `moderate-check` → kiểm tra danh sách từ khóa cấm (`malware`, `phishing`, `evil`), trả `{code, target, safe}`.
  - [ ] **Step 2:** Triển khai Go/ASL giữ Retry/Catch/updateItem gốc theo kiến trúc5.3; thêm runbook manual input `{code,target}`, safe/reject/retry/Fail, tạo META hợp lệ trước và không xóa song song vì ASL có thể upsert. Không nối moderation vào create/redirect.
  - [ ] **Step 3:** Chạy Go race/vet, build 8 binaries và mở rộng CI; execution history Step Functions thật nằm trong AWS E2E khi được phép.
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
  - [ ] **Step 3:** Chạy cổng Terraform fmt/init-backend=false/validate/test ở cuối kế hoạch, cập nhật CI và AWS E2E scenarios.
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
  - Create: `frontend/src/app/login/page.tsx` (nút Tiếp tục với Google qua Cognito Code + PKCE)
  - Create: `frontend/src/app/auth/callback/page.tsx` (OAuth2 PKCE Callback)
  - Create: `frontend/src/lib/auth.test.ts`
  - Create: `tests/e2e/frontend.spec.ts` (mở rộng theo Task 5.2/5.3)
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Khởi tạo Next.js với output export/trailingSlash/unoptimized; chọn và pin phiên bản đang hỗ trợ, package manager/lockfile và test runner, mở rộng CI.
  - [ ] **Step 2:** Chỉ Google federation qua Cognito `/oauth2/authorize?identity_provider=Google`, Code + PKCE S256; test state/nonce/verifier, redirect allowlist, callback lỗi/thiếu/lặp code, token expiry/logout, lấy Cognito access token và từ chối ID/Google token ở API. Không form mật khẩu, signup riêng hay password E2E flow; không dùng password demo/innerHTML. Public config build lấy output môi trường, validation không chấp nhận endpoint placeholder.
  - [ ] **Step 3:** Xây dựng Landing, login Google và callback theo Hand-Drawn / Doodle & Sketch UI: nền giấy sáng, nét viền/chú thích vẽ tay tiết chế, thứ bậc đọc rõ, focus/contrast/reduced-motion, responsive và loading/error states; không dùng ảnh chụp bình luận làm asset.
  - [ ] **Step 4:** Chạy frontend lint/typecheck/test/build theo cổng kiểm tra; kiểm chứng output tĩnh và callback không cần Node runtime.
  - [ ] **Step 5:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 5.1.

#### Task 5.2: Dashboard, My Links, Tích Hợp HTTP API Thật & Native WebSocket Realtime Hook
- **Branch:** `feat/phase-05-task-02-dashboard-api-and-websocket`
- **Files:**
  - Create: `frontend/src/lib/api.ts`
  - Create: `frontend/src/hooks/useRealtimeClicks.ts`
  - Create: `frontend/src/hooks/useRealtimeClicks.test.ts`
  - Create: `frontend/src/features/links/CreateLinkForm.tsx`
  - Create: `frontend/src/features/links/LinksTable.tsx`
  - Create: `frontend/src/app/dashboard/page.tsx`
  - Create: `frontend/src/app/links/page.tsx`
  - Modify: `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Xây dựng `api.ts` gọi trực tiếp `POST /links`, `GET /links`, `DELETE /links/{code}` kèm header `Authorization: Bearer <token>` và xử lý rõ ràng các trạng thái lỗi `400`, `401`, `404`, `429`, `500`.
  - [ ] **Step 2:** Hook WebSocket native kết nối theo code, reconnect exponential backoff, nạp lại GET links khi reconnect, dọn socket khi delete/unmount; hợp nhất snapshot/push bằng max để count không lùi. Test disconnect/reconnect/out-of-order, không polling giả realtime; hiển thị trạng thái kết nối và giới hạn push best-effort.
  - [ ] **Step 3:** Hoàn thiện trang `/dashboard` và `/links` theo cùng hệ thiết kế Doodle & Sketch (tạo link, copy short link, nút mở link/click thử, xóa link, tổng số clicks nhảy realtime, trạng thái kết nối WebSocket); ưu tiên dữ liệu và trạng thái dễ đọc hơn trang trí.
  - [ ] **Step 4:** Chạy frontend lint/typecheck/test/build; kiểm tra `frontend/out/` và E2E với HTTP harness thật khi local, AWS API thật khi đã được phép.
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
  - [ ] **Step 2:** CloudFront cache `_next/static/*` immutable1year, HTML TTL ngắn; viewer-request rewrite `/dashboard` → `/dashboard/index.html`, bảo toàn callback query. Missing route/asset non-200 (403 hoặc404 private S3), không catch-all200; không cấp ListBucket rộng hay che OAC error để ép404. Test rewrite và E2E routes/assets/missing/auth callback.
  - [ ] **Step 3:** Cập nhật CORS HTTP API và callback/logout URLs Cognito theo domain CloudFront thật (`/auth/callback/`); giữ Google OAuth redirect URI trỏ Cognito `/oauth2/idpresponse`, không trỏ CloudFront.
  - [ ] **Step 4:** Chạy cổng Terraform fmt/init-backend=false/validate/test ở cuối kế hoạch, cập nhật CI và frontend E2E.
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
  - [ ] **Step 1:** Observability giữ 2 alarms nguồn Sum>=1/60s/1period/notBreaching; bổ sung DLQ alarm và metrics ClickPublishFailed/ClickDiscardedDeleted như delta đích. Dashboard hiển thị Invocations/Errors/Throttles, Duration p50/p99, backlog/DLQ và LinkResolved. Đặt retention Logs, tags; SNS subscription tùy chọn do người dùng cấu hình/confirm, không tự gửi email. SLO chỉ mục tiêu đến khi có evidence.
  - [ ] **Step 2:** CodeDeploy LambdaCanary10Percent5Minutes dùng ResolveErrorAlarm (thêm throttle alarm là delta đích), auto rollback; tiếp nhận alias live đã tạo Task2.2 với ignore_changes function_version/routing_config. API integration và permission phải dùng alias, lấy CurrentVersion từ alias thật và target published version mới. Một writer, same version no-op, đợi deployment success/rollback kết thúc, không manual alias reset khi lỗi.
  - [ ] **Step 3:** Chạy cổng Terraform fmt/init-backend=false/validate/test ở cuối kế hoạch, cập nhật CI và canary E2E.
  - [ ] **Step 4:** Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 6.1.

#### Task 6.2: GitHub Actions CI/CD (AWS OIDC, Canary Deploy, S3/CloudFront Sync), k6 Load Test & Tài Liệu Vận Hành E2E
- **Branch:** `feat/phase-06-task-02-cicd-k6-and-runbook`
- **Files:**
  - Modify: `.github/workflows/ci.yml` (đã có từ Task 1.1)
  - Create: `.github/workflows/deploy.yml`
  - Create: `scripts/deploy-canary.sh`
  - Create: `tests/load/redirect.k6.js`
  - Modify: `tests/e2e/aws-flow.ps1`, `tests/e2e/frontend.spec.ts`
  - Create: `docs/e2e-verification-and-operations.md`
  - Modify: `README.md`, `Makefile`, `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`
- **Steps:**
  - [ ] **Step 1:** Hoàn thiện CI đã mở rộng qua các phase: Go race/vet, build8ZIP, Next lint/typecheck/test/static build, Terraform fmt/init-backend=false/validate/test. Không đợi Phase6 mới chạy CI.
  - [ ] **Step 2:** CD OIDC dùng id-token:write/contents:read, role-to-assume bootstrap output, exact aud/sub, protected environment và manual workflow_dispatch; serialized concurrency, không apply PR/fork không tin cậy. Pin actions lúc triển khai. Bootstrap operator chạy riêng khi được phép. Build Lambda → saved Terraform plan được review → apply app → lấy API/WS/Cognito/CF outputs → public config → Next static build → canary success → sync hashed assets trước HTML → invalidate. Build placeholder bị chặn; rollback/failure dừng publish frontend. Tuân thủ alias ownership ở kiến trúc6.
  - [ ] **Step 3:** k6 dùng redirects:0, mặc định MODE=smoke 2VUs/10s với sleep1s sau mỗi request (tối đa khoảng2RPS); MODE=stress ramp40 chỉ khi được yêu cầu rõ, báo cả successful throughput và failed/throttled, không dùng benchmark Node làm baseline Go.
  - [ ] **Step 4:** Tạo E2E scripts/runbook theo ma trận bên dưới: service ESM/DLQ, SFN execution history, OIDC negative trust, canary success và fault-induced rollback trong sandbox được phép. Ghi retention/chi phí/cleanup và redrive có đối soát vì DLQ14d dài hơn dedupe24h; không blanket replay dữ liệu cũ.
  - [ ] **Step 5:** Chạy toàn bộ lệnh kiểm tra (`make check`), cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md` và mở Pull Request riêng cho Task 6.2.

---

## Cổng Kiểm Tra Và Bằng Chứng Nghiệm Thu

Các lệnh dưới đây là hợp đồng đầu ra cho task tương ứng; file test/harness chưa tồn tại ở Phase0. Unit mock chỉ ở ranh giới SDK/clock/network; integration chạy DynamoDB Local/container hoặc AWS được phép. Local không chứng minh JWT authorizer, IAM, ESM, DLQ hay canary thật. Mỗi task mở rộng `.github/workflows/ci.yml` cùng implementation.

- Backend: `go -C backend test -race ./...`, `go -C backend vet ./...`; integration `go -C backend test -tags=integration ./internal/repository/...` với `DYNAMODB_ENDPOINT` trỏ DynamoDB Local/container.
- Frontend: `pnpm --dir frontend lint`, `pnpm --dir frontend typecheck`, `pnpm --dir frontend test`, `pnpm --dir frontend build`; kiểm `out/` có HTML các route, không đòi Node runtime.
- Terraform: `terraform fmt -check -recursive infrastructure/terraform`; mỗi root bootstrap/dev chạy `terraform -chdir=<root> init -backend=false`, `validate`, `test` (tests chỉ plan/mock provider trong CI; không AWS apply). Assertions module tập trung qua root tests.
- Mỗi lần kiểm ghi command/environment/exit code, assertion và đường dẫn artifact; không ghi token/password. AWS tests giữ **pending** đến khi có quyền triển khai rõ ràng, không thay bằng kết quả unit.

| Task / file cần tạo hoặc mở rộng | Ca kiểm thử và lệnh | Môi trường / bằng chứng |
| --- | --- | --- |
| 1.1, 2.1: `backend/internal/httpapi/handler_test.go`, `backend/internal/service/link_service_test.go` | Go race/vet: unauth401; invalid URL, body null/nonstring400; collision đúng5 lần tổng; create201, resolve301/404; A list chỉ của A, B delete A404 | Unit boundary mocks; assertion status/body/Location/số lần gọi |
| 2.1: `backend/internal/events/eventbridge_test.go` | `go -C backend test -race ./internal/events/...`: HTTP200 nhưng failed1, nonretryable, retry success/exhausted, entry thiếu/sai; deadline/retry budget | Unit SDK boundary; assert không301/LinkResolved khi chưa có EventId, ClickPublishFailed khi lỗi |
| 2.1: `backend/internal/repository/dynamodb_integration_test.go`, `backend/tools/local-http/main.go` | Integration command trên; `go -C backend run ./tools/local-http -config ../scripts/local-http/config.example.json`; gọi 4 routes qua harness | DynamoDB Local/container và handler thật; owner condition, collision, pagination, timestamp ordering, response; docs/local-testing.md ghi cách cấp test identity/publisher endpoint local, không giả IAM |
| 2.2: `tests/e2e/aws-flow.ps1` | `pwsh tests/e2e/aws-flow.ps1 -Scenario Core`: thiếu JWT401, Google/ID token bị từ chối, Cognito access token đúng scope được nhận; A tạo/list, B không thấy và delete A404, invalid/null/nonstring400, redirect301 với redirects disabled | AWS được phép; hai tài khoản Google thử nghiệm đăng nhập qua Cognito browser federation để lấy phiên, actual authorizer/IAM/DynamoDB; HTTP evidence và cleanup chỉ link do test tạo, token không ghi log |
| 3.1: `backend/internal/service/aggregator_service_test.go`, repository integration | Go race + integration: duplicate, mixed/nil/empty reasons, conflicts, delete race, consistent-read lỗi, TTL qua nhưng marker còn và marker removal/replay | DDB Local transaction; kiểm META/STAT/marker sau từng bước; chỉ message lỗi trong BatchItemFailures |
| 2.2, 3.2: `tests/e2e/aws-flow.ps1` | `pwsh tests/e2e/aws-flow.ps1 -Scenario Pipeline`: resolve publish → bus rule → SQS → ESM; gây lỗi xử lý để kiểm maxReceive3 và DLQ | AWS sandbox được phép; ARN config, message/event IDs, receive observations, DLQ evidence; unit không thay thế service test |
| 4.1, 5.2: `backend/internal/events/ws_pusher_test.go`, `frontend/src/hooks/useRealtimeClicks.test.ts`, `tests/e2e/frontend.spec.ts` | Go events tests + pnpm test; `pnpm --dir frontend exec playwright test ../tests/e2e/frontend.spec.ts`: push410 cleanup, non410/Get/Query/cleanup error, reconnect reload, out-of-order max | Unit faults; browser E2E với API/WS thật khi được phép, record commit không retry vì notifier |
| 4.1, 4.2: `tests/e2e/aws-flow.ps1` | `pwsh tests/e2e/aws-flow.ps1 -Scenario Moderation`: safe/reject/retry/MarkError; tạo META hợp lệ trước, không xóa song song | AWS sandbox; execution history chứng minh Retry/Catch/Fail và status, không chỉ test hàm blocklist |
| 5.1, 5.3: `frontend/src/lib/auth.test.ts`, frontend E2E | pnpm test/build + Playwright: nút Google duy nhất, state/nonce/verifier/callback errors, logout; kiểm desktop/mobile, focus/contrast/reduced-motion; CF `/dashboard`, `/auth/callback/?code=...&state=...`, assets, missing non-200 (403/404) | Unit auth và static build local; Google→Cognito→CloudFront callback thật khi deploy được phép, chụp response/status và không lưu token/secret |
| 2.2, 6.2: `bootstrap/tests/trust.tftest.hcl`, `tests/e2e/aws-flow.ps1` | Terraform test; `pwsh tests/e2e/aws-flow.ps1 -Scenario OIDC`: đúng repo/ref/environment được assume, sai aud/sub/ref/fork bị từ chối | Unit/plan assertions + workflow sandbox được phép; STS denial/success evidence, không in JWT |
| 6.1, 6.2: `scripts/deploy-canary.sh`, `tests/e2e/aws-flow.ps1` | `pwsh tests/e2e/aws-flow.ps1 -Scenario Canary`: same-version no-op, rollout thành công và fault-induced alarm rollback | AWS sandbox được phép; deployment ID/history, alarm transition, alias trước/sau, qualified API invocation; frontend không publish khi fail |
| 6.2: `tests/load/redirect.k6.js` | `k6 run -e MODE=smoke tests/load/redirect.k6.js` (base URL/code từ cấu hình); redirects0 | AWS được phép hoặc local harness; tách status301/429/5xx, failed và successful RPS; stress cần yêu cầu riêng |

Task sở hữu hành vi chịu trách nhiệm thêm test/file trong ma trận ngay khi triển khai; Task6.2 nối các kịch bản AWS thành runbook thống nhất. Phase0 chỉ chạy kiểm tra diff/liên kết/ví dụ JSON, không chạy các lệnh app/Terraform chưa có implementation.
