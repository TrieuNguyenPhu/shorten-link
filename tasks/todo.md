# Checklist triển khai theo phase

Đọc cùng [plan.md](plan.md) và [đối chiếu series](../docs/series-review.md). **Phase 02A hoàn tất offline; backend Phase 02B đã deploy tại Sydney và vượt 20/20 smoke checks**. CloudFront đang chờ AWS xác minh account nên Phase 02 chưa hoàn tất; xem [bằng chứng AWS](../docs/phase-02b-verification.md). CI đã được định nghĩa/kiểm tra tĩnh, chưa chạy trên GitHub runner. Các phase sau chưa thực hiện.

Mỗi task gồm tối đa vài file chính, kèm test/config liên quan; nếu diff vượt một phiên review thì chia nhỏ trước khi code. Lệnh Go bên dưới chạy từ `services/lambda`; lệnh pnpm/Terraform chạy từ root. Không chạy các kiểm tra cloud trước khi được yêu cầu dùng AWS.

## Phase 01 — Giữ baseline

- [x] MVP local đã bàn giao: create/get/list, redirect, aliases, expiry, enable/disable, DynamoDB Local, frontend static export và tests. Xem [verification](../docs/verification.md); không coi đây là lần chạy test mới.

## Phase 02A — Cloud-ready, chưa provision

### 02.1 — Native Lambda transport

**Phụ thuộc:** Phase 01. **Scope:** M.

**Files dự kiến:** `internal/adapters/lambdaapi/handler.go`, `handler_test.go`, `cmd/create-link/main.go`, `cmd/resolve-link/main.go`.

- [x] API Gateway v2 events gọi use cases hiện có qua `aws-lambda-go`; body/path/base64-body được validate và response giữ contract create/redirect hiện tại.
- [x] Test thành công, malformed JSON/code, missing/disabled/expired link, collision và downstream error; không khởi động `net/http` server trong Lambda.

**Nghiệm thu:** `go test ./internal/adapters/lambdaapi -run TestHTTPV2 -v`, `go test ./...`, `go build ./...`. Đối chiếu fixtures với HTTP local; tên test là mục tiêu sẽ thêm.

### 02.2 — Management adapter và AWS bootstrap

**Phụ thuộc:** 02.1. **Scope:** M.

**Files dự kiến:** `cmd/manage-links/main.go`, `internal/awsconfig/config.go`, `config_test.go`, Lambda adapter/tests của 02.1.

- [x] List/get/enable qua native transport, tương đương local adapter; SDK client tái sử dụng qua invocation, truyền context từng request.
- [x] Cloud configuration dùng region/resource names và execution-role credentials; local entrypoint vẫn loopback/dummy credentials. Test không đọc profile hoặc gọi metadata service thật.

**Nghiệm thu:** AWS config tests với client/config fixtures, contract tests cho management, `go test ./...`, `go vet ./...`.

### 02.3 — Build artifact lặp lại được

**Phụ thuộc:** 02.2. **Scope:** S.

**Files dự kiến:** `local/build-lambdas.mjs`, `local/build-lambdas.test.mjs`, `Makefile`, `.gitignore`.

- [x] Mỗi entrypoint có Linux/arm64 binary `bootstrap` trong ZIP riêng; flags/runtime tương thích; ZIP giữ executable permission cả khi build từ Windows.
- [x] Artifact có digest, không chứa env/secrets/source thừa; build output được ignore. Không deploy trong lệnh build.

**Nghiệm thu:** test cấu trúc/mode ZIP và ELF architecture bằng self-check; build từ host hiện tại và Linux CI.

### 02.4 — Terraform backend và skeleton dev

**Phụ thuộc:** 02.3. **Scope:** M.

**Files dự kiến:** `infra/terraform/bootstrap/main.tf`, `infra/terraform/dev/{versions,backend,variables,outputs}.tf`; lockfiles/config tối thiểu.

- [x] Pin Terraform/provider; state bucket bootstrap tách application state; có encryption/versioning và S3 `use_lockfile`, không thêm DynamoDB lock table mới.
- [x] Variables cho region/environment/tags; không chứa tài khoản/bucket bí mật hoặc credentials mẫu có thể dùng thật. Có hướng dẫn bootstrap nhưng chưa chạy.

**Nghiệm thu:** `terraform -chdir=infra/terraform/dev fmt -check`, `init -backend=false`, `validate`; kiểm tương tự root bootstrap. Chỉ registry downloads, không truy cập account AWS.

### 02.5 — DynamoDB, Lambda và IAM definitions

**Phụ thuộc:** 02.4. **Scope:** M.

**Files dự kiến:** `infra/terraform/dev/{data,compute,iam,logging}.tf`, `docs/aws-development.md`.

- [x] DynamoDB schema khớp adapter; resource names/env/artifact paths rõ ràng; Lambda dùng runtime/architecture của 02.3 và quyền từng nhóm tác vụ.
- [x] Structured logs, retention, gateway/Lambda request correlation và baseline failure metrics; production data có deletion protection/PITR theo môi trường, không tự destroy table/state.

**Nghiệm thu:** Terraform validate/mock-provider assertions về IAM/resource config; artifact build và local Go integration tests. Chưa đánh dấu IAM đã hoạt động trên AWS.

### 02.6 — HTTP API routing và bảo vệ giai đoạn chưa Cognito

**Phụ thuộc:** 02.5. **Scope:** S/M.

**Files dự kiến:** `infra/terraform/dev/api.tf`, Lambda adapter tests, `docs/aws-development.md`.

- [x] Giữ URL `/api/links`, `/api/links/{code}`, `/r/{code}`, `/health`; payload v2, Lambda invoke permissions theo route, CORS origin tường minh và throttle cấu hình được.
- [x] Management routes dùng `AWS_IAM` trước Phase 03; public redirect không cần login. Không có owner giả, token dev cố định hoặc public unauthenticated mutation.

**Nghiệm thu:** Terraform assertions về authorization types/integrations; native route-contract tests. Ghi rõ frontend cloud chưa quản lý link bằng browser cho đến Phase 03.

### 02.7 — Static hosting definitions

**Phụ thuộc:** 02.6. **Scope:** M.

**Files dự kiến:** `infra/terraform/dev/{hosting,routing}.tf`, CloudFront Function source/test, `docs/aws-development.md`.

- [x] S3 private/OAC; CloudFront static origin và behaviors API/redirect cache-disabled; Authorization và các request fields cần thiết được forward đúng.
- [x] `/` và `/dashboard/` map đúng exported HTML; API path/asset không bị rewrite sai. Build-time API URL được cấu hình theo environment.

**Nghiệm thu:** rewrite self-check cho routes/assets/API; `pnpm build`; Terraform validate và assertions về OAC/cache behavior.

### 02.8 — CI kiểm tra local trên máy sạch

**Phụ thuộc:** 02.7. **Scope:** S/M.

**Files dự kiến:** `.github/workflows/ci.yml`, Playwright/local tooling khi cần, README development commands.

- [x] PR chạy Go format/vet/tests, DynamoDB Local integration, pnpm frozen install/lint/typecheck/static build và Chromium E2E; artifact build được kiểm.
- [x] Terraform fmt/backend-disabled init/validate; không deploy, không assume AWS role, không đưa credentials cho PR không tin cậy.

**Nghiệm thu:** workflow chạy thành công trên runner sạch khi có repository remote; nếu chưa có remote thì báo giới hạn và chỉ ghi kết quả kiểm local đã chạy.

### Checkpoint 02A

- [x] Local baseline giữ xanh; Lambda payload contract và packaging được chứng minh.
- [x] Terraform definitions, IAM, static routing và CI diff đã review; chưa apply.
- [x] Bàn giao artifact/diff và dừng trước thao tác AWS cho đến khi có yêu cầu triển khai.

## Phase 02B — AWS core dev, chỉ khi được yêu cầu

### 02.9 — Provision dev và smoke test

**Phụ thuộc:** Checkpoint 02A và yêu cầu triển khai AWS. **Scope:** S thay đổi code, có thao tác cloud.

**Files dự kiến:** `docs/aws-development.md`, cấu hình dev không bí mật, smoke test fixtures/scripts tối thiểu.

- [x] Xác nhận account `654822777701`, Sydney `ap-southeast-2`, ngân sách 4 USD/tháng; bootstrap state và review saved plan trước apply. State/credentials được loại khỏi Git.
- [ ] Static page/deep link hoạt động qua CloudFront; SigV4 developer có thể create/manage, caller không có quyền bị chặn; public redirect đúng 302/404/410. Direct S3 access bị từ chối.
- [ ] Kiểm log/metrics và resource policies thực tế; có hướng dừng tài nguyên dev mà không xóa dữ liệu ngoài scope.

Hướng dẫn [tạm dừng/khôi phục dev](../docs/aws-dev-pause-resume.md) đã hoàn tất, preflight read-only đã kiểm tra; chưa diễn tập outage. ACM đã được cấp, cấu hình dev đã gắn domain/certificate; apply vẫn bị AWS chặn xác minh account. Bucket policy và CloudFront chưa thể nghiệm thu.

Backend log correlation, metric filter, IAM negative checks và private S3 đã kiểm tra. AWS đã tạo 54/56 resources của dev stack; CloudFront và bucket policy còn chờ account verification. [Bản nháp AWS Support](../docs/aws-support-cloudfront.md) đã chuẩn bị, chưa gửi. Giữ hai mục nghiệm thu trên chưa hoàn tất cho đến khi kiểm được CloudFront, domain và frontend thực tế.

**Nghiệm thu:** cloud smoke evidence, permission-negative tests, inventory/cost tags; ghi đầy đủ giới hạn. Không tuyên bố có browser auth/ownership trước Phase 03.

## Phase 03 — Authentication & Link Management

### 03.1 — Cognito và HTTP API JWT scopes

**Phụ thuộc:** 02.9. **Scope:** M.

**Files dự kiến:** `infra/terraform/dev/auth.tf`, `api.tf`, `contracts/openapi.yaml`, `docs/authentication.md`.

- [ ] User pool/public client và callback/logout URLs tường minh; authorization code + PKCE; management routes yêu cầu access-token scopes, redirect vẫn public.
- [ ] Thiếu token, token sai issuer/audience, hết hạn hoặc thiếu scope bị từ chối; default gateway errors được mô tả trong contract.

**Nghiệm thu:** test bằng Cognito dev thật khi đã có quyền triển khai; xác nhận request bị gateway chặn không tới Lambda.

### 03.2 — Actor và ownership trong data access

**Phụ thuộc:** 03.1. **Scope:** M, chia domain/adapter thành hai diff nếu cần.

**Files dự kiến:** domain Link/Actor, application use cases/tests, repository port, DynamoDB adapter/integration tests.

- [ ] Actor lấy từ trusted claims; owner index `USER#sub`; metadata/list/status-update đều scope theo actor; mutation condition kiểm ownership tại storage.
- [ ] Test A/B và cursor tampering; không tự nhận dữ liệu ownerless. Cloud table tách local; cần mapping riêng nếu giữ fixture cloud cũ.

**Nghiệm thu:** Go unit/integration tests, các request A/B thực tế; không có đường truy cập metadata của người khác qua code đoán được.

### 03.3 — Frontend login và session

**Phụ thuộc:** 03.2. **Scope:** M.

**Files dự kiến:** frontend auth client/provider, static callback page, workspace/session UI, `src/lib/api.ts`.

- [ ] Đăng ký/xác minh/đăng nhập/logout bằng Cognito; PKCE, state/nonce và token refresh dùng thư viện được duy trì thay vì tự viết crypto flow.
- [ ] API gắn access token; 401/403 và gateway error shape xử lý rõ; UI không xem ẩn nút là authorization. Token không đi vào logs hoặc bundle.

**Nghiệm thu:** browser login→create→logout→blocked API; expiry/refresh/error states; `pnpm lint`, `pnpm typecheck`, `pnpm build` vẫn static.

### 03.4 — Edit và delete có version control

**Phụ thuộc:** 03.2–03.3. **Scope:** M mỗi diff backend/frontend.

**Files dự kiến:** application/domain/port, DynamoDB adapter, OpenAPI contract; form/library/API client cập nhật trong diff nối tiếp.

- [ ] Owner sửa destination/expiry/enabled và xóa; optimistic version conflict không ghi đè thay đổi đồng thời. Validation URL giữ nguyên, không fetch destination.
- [ ] Xóa tạo tombstone tối thiểu và bỏ owner-index keys; không tái cấp code cũ. UI có xác nhận xóa, loading/error/success và hiển thị conflict để tải lại dữ liệu.

**Nghiệm thu:** tests A/B cho mọi endpoint; concurrent edits; old redirect sau delete; browser manage E2E và static build.

### Checkpoint 03

- [ ] Hai user thật tách biệt ở list/get/edit/status/delete; URL local không được tự publish.
- [ ] Browser auth hoàn chỉnh; public redirect vẫn hoạt động; cloud/local transport contracts thống nhất ở phần nghiệp vụ chung.

## Phase 04 — Event-driven Analytics

### 04.1 — Event contract và publisher

**Phụ thuộc:** Checkpoint 03. **Scope:** M.

**Files dự kiến:** `contracts/link-clicked.v1.schema.json`, application redirect tests, `ports/events.go`, EventBridge adapter/tests.

- [ ] Application event ID ổn định qua publish retries; event có schemaVersion/code/owner/occurredAt UTC; chỉ GET thành công phát event, không phát cho HEAD/404/410.
- [ ] Chốt redirect-first hoặc durable-delivery policy trước code; test timeout, SDK error, partial entry failure và cấu hình bus sai bằng smoke test. Không fire-and-forget sau return.

**Nghiệm thu:** producer unit/adapter tests; latency budget và failure metric được ghi trong contract/runbook.

### 04.2 — EventBridge → SQS và failure paths

**Phụ thuộc:** 04.1. **Scope:** M.

**Files dự kiến:** `infra/terraform/dev/events.tf`, queue-policy assertions, event fixtures, docs event flow.

- [ ] Bus/rule/target/queue policy đúng source ARN; consumer queue + DLQ và target-delivery DLQ có quyền/retention rõ ràng.
- [ ] Visibility timeout theo Lambda timeout/batch window; maxReceiveCount, concurrency và batch size theo tài liệu/quota; bật partial batch response.

**Nghiệm thu:** Terraform validate/assertions; một event hợp lệ qua toàn tuyến, delivery failure được quan sát riêng với consumer failure.

### 04.3 — Transactional aggregator

**Phụ thuộc:** 04.2. **Scope:** M, tách persistence và SQS transport nếu vượt scope.

**Files dự kiến:** application analytics/tests, DynamoDB analytics adapter/integration tests, `cmd/analytics/main.go`, SQS adapter/tests.

- [ ] Một transaction ghép dedupe marker, total và UTC-day counters, cùng điều kiện không phục hồi link đã xóa; event timestamps/schema được validate.
- [ ] Duplicate chỉ ACK khi xác nhận marker conflict; transaction conflict/throttle/failure retry. Partial batch trả SQS message identifiers, dedupe dùng application event ID.
- [ ] N unique events tăng N; duplicates/concurrent duplicates tăng một; lỗi trước commit không để marker đơn độc hoặc counter sai.

**Nghiệm thu:** Go unit + real DynamoDB transaction tests, mixed-batch fixtures và concurrent counter test.

### 04.4 — Analytics API và dashboard

**Phụ thuộc:** 04.3. **Scope:** M mỗi backend/frontend slice.

**Files dự kiến:** analytics read use case/route/tests, OpenAPI; dashboard chart/component/API client.

- [ ] Owner xem total và dữ liệu ngày trong khoảng giới hạn, query phân trang; không scan, không đọc metadata/metrics của user khác.
- [ ] Chart thật, empty/loading/error states và dữ liệu eventual consistency; không gọi tổng request là unique visitors.

**Nghiệm thu:** Go read/ownership tests; browser known-clicks→chart, date bounds, UTC day boundary và static export.

### 04.5 — Retry, DLQ và redrive rehearsal

**Phụ thuộc:** 04.4. **Scope:** S/M.

**Files dự kiến:** fault fixtures/tests, `docs/runbooks/analytics.md`, alarm definitions.

- [ ] Poison event vào DLQ; lỗi transient phục hồi; redrive giữ application ID và không cộng trùng. Dedupe/replay horizon thực thi đúng policy 30 ngày hoặc giá trị đã chốt.
- [ ] Runbook phân biệt delivery DLQ, consumer DLQ và producer publish failure; có evidence và metric cho từng chặng.

**Nghiệm thu:** diễn tập lỗi trên dev được phép; đối chiếu expected totals, retries và queue age. Không purge dữ liệu người dùng để làm test.

### Checkpoint 04

- [ ] Tính đúng của counters, at-least-once handling và giới hạn có thể mất click đã được ghi rõ, có tests/evidence.
- [ ] UI analytics và runbook hoạt động trước khi thêm realtime.

## Phase 05A — Realtime

### 05.1 — Ticket và connection ownership

**Phụ thuộc:** Checkpoint 04. **Scope:** M mỗi ticket/connection slice.

**Files dự kiến:** authenticated ticket endpoint/use case/tests, WS authorizer/connect/disconnect handlers, connection persistence/tests, WS Terraform.

- [ ] JWT-protected endpoint cấp ticket dùng một lần gắn owner/code; `$connect` kiểm ticket và connection session deadline. Không truyền JWT dài hạn bằng query string.
- [ ] B không thể nhận ticket/subscribe link của A; ticket replay/expired bị chặn; TTL không thay kiểm hết hạn tại runtime.

**Nghiệm thu:** ownership và ticket tests; real dev handshake/replay/session-expiry checks.

### 05.2 — Notification và browser recovery

**Phụ thuộc:** 05.1. **Scope:** M.

**Files dự kiến:** Management API push adapter/tests, aggregator integration, dashboard WebSocket hook/status UI, realtime E2E.

- [ ] Sau commit có notification đúng subscribers, query subscriber có pagination; 410 dọn connection, lỗi khác được ghi nhận mà không làm cộng counter lại.
- [ ] Client debounce/refetch authorized snapshot, reconnect có backoff, refresh khi nhận lại kết nối; mất/trễ notification không làm UI giữ dữ liệu sai vô hạn.

**Nghiệm thu:** click→UI update, disconnect/reconnect, failed push và cross-user leakage tests; đo freshness theo target đã chốt.

### Checkpoint 05A

- [ ] Realtime có auth, cleanup và recovery; HTTP snapshot vẫn là đường đọc số liệu có quyền.

## Phase 05B — Moderation

### 05.3 — State model và checker

**Phụ thuộc:** Checkpoint 03; lên lịch sau 05A để giữ roadmap tuyến tính. **Scope:** M.

**Files dự kiến:** domain/application validation/status tests, checker adapter/entrypoint, moderation contract và UI status mapping.

- [ ] Pending/approved/rejected/error tách enabled/deleted/expiry; create/edit URL vào pending; resolve yêu cầu approved.
- [ ] Policy demo được ghi đúng giới hạn; reputation provider chỉ tích hợp sau khi chọn. Server không fetch arbitrary destination; input policy và provider response được validate.

**Nghiệm thu:** redirect truth-table tests cho mọi state; pending/rejected/error UI và source URL privacy checks.

### 05.4 — Step Functions và end-to-end workflow

**Phụ thuộc:** 05.3. **Scope:** M từng workflow/integration diff.

**Files dự kiến:** ASL workflow, Terraform moderation IAM/resources, starter adapter/tests, create/edit integration, retry action.

- [ ] Standard workflow có Choice/Retry/Catch, execution identity theo code/version; start failure giữ link không active và có retry rõ ràng.
- [ ] Approval conditional theo version; workflow cũ không approve URL mới, không bỏ disable, không resurrect deleted link. Retry/re-execution có test idempotency.

**Nghiệm thu:** approve/reject/transient error/exhausted retry/start-failure/stale-result tests, dev execution history và UI polling trạng thái.

### Checkpoint 05B

- [ ] URL chỉ redirect sau policy approval; bất kỳ provider/workflow failure nào cũng không mặc nhiên approve.

## Phase 06 — DevOps & Production Readiness

### 06.1 — Observability, SLO và runbooks

**Phụ thuộc:** Checkpoint 05B. **Scope:** M mỗi instrumentation/alarms diff.

**Files dự kiến:** Go observability bootstrap, Terraform monitoring, `docs/runbooks/`, SLO document.

- [ ] OpenTelemetry → X-Ray, structured log correlation và metrics; không bê middleware TypeScript. Sampling/log retention kiểm soát được.
- [ ] SLI gồm gateway/client outcomes và queue freshness; alarm có runbook cho throttle/5xx, publish failures, queue age/DLQ, moderation và push failures. Không dùng per-code metric dimensions.

**Nghiệm thu:** trace xuyên các chặng hỗ trợ, alarm fault-injection có action evidence; đánh dấu phần AWS gateway không tự trace được thay vì giả định.

### 06.2 — OIDC deployment pipeline

**Phụ thuộc:** 06.1 và CI 02.8. **Scope:** M.

**Files dự kiến:** `.github/workflows/deploy.yml`, Terraform OIDC/IAM, release scripts, deployment runbook.

- [ ] OIDC audience/subject khớp repo/environment thực tế; deploy role hẹp; PR không tin cậy không được lấy quyền. Không lưu access key dài hạn.
- [ ] CI artifact cố định bằng digest, reviewed infra plan, serialized deployment và environment gate; production deployment chỉ khi được yêu cầu.

**Nghiệm thu:** negative trust-policy checks và dev deployment từ GitHub; không ghi “CD passed” nếu mới chạy workflow local hoặc chưa có remote.

### 06.3 — Canary và rollback đã diễn tập

**Phụ thuộc:** 06.2. **Scope:** M.

**Files dự kiến:** CodeDeploy/version/alias resources, release scripts/hooks, rollback test và runbook.

- [ ] API integrations gọi stable alias; canary 10%/5 phút là cấu hình khởi điểm, có synthetic requests khi tải thấp và alarm trên đúng version/alias.
- [ ] Good release tiến lên; deliberately failing release rollback thật; Terraform sau đó không reset alias/routing ngoài ý muốn. Frontend cũ/backend mới và ngược lại tương thích trong rollout.

**Nghiệm thu:** lưu deployment ID, alarm/hook result, alias before/after và smoke outcomes; kiểm lại Terraform drift/plan. Canary success không thay rollback test.

### 06.4 — Hardening và recovery

**Phụ thuộc:** 06.3. **Scope:** M mỗi security/recovery diff.

**Files dự kiến:** IAM/header/WAF config nếu cần, security tests, backup/restore và incident runbooks.

- [ ] Review IDOR, session handling, CORS, CSP, secrets/log redaction, least privilege và abuse limits. Nếu có WAF, chặn và kiểm direct-origin bypass thực tế; không coi OAC là bảo vệ API.
- [ ] Restore DynamoDB ra table dev riêng rồi kiểm đọc; schema/event upgrades rollback được; quy trình cleanup không xóa state hoặc data thật ngoài scope.

**Nghiệm thu:** permission-negative tests, restore drill và sáu trụ cột Well-Architected, gồm Sustainability; ghi giới hạn bảo mật còn lại.

### 06.5 — Load và cost acceptance

**Phụ thuộc:** 06.4. **Scope:** S/M.

**Files dự kiến:** `tests/load/redirect.js`, scenario fixtures, cost estimate và benchmark report.

- [ ] k6 `redirects: 0`; đo cả một hot link và nhiều link; nominal test đạt success/latency target đã chốt; overload test báo riêng 429/5xx, không dùng latency rejection làm bằng chứng throughput.
- [ ] Đối chiếu quotas/CloudWatch/queue lag và cost estimate có frontend, transactions, WebSocket, logs/traces; budget/tag/retention theo môi trường. Không mặc định free tier hoặc số quota của bài viết.

**Nghiệm thu:** raw results, client/server correlation, bottleneck analysis, cost model theo giá tại ngày đo; test không đi theo destination bên ngoài hoặc gây tải ngoài dev scope.

### Checkpoint hoàn thành project

- [ ] Functional/auth/event/realtime/moderation E2E có bằng chứng, không chỉ cấu hình tồn tại.
- [ ] Có CI/CD, canary/rollback, alarm/runbooks, restore, load/cost report và giới hạn đã công bố.
- [ ] Chỉ kết luận production-ready cho các acceptance đã thực sự kiểm; không tự tiếp tục thêm custom domain, multi-region hoặc tính năng mới.
