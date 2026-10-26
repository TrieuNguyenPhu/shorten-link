# Phase 0 — Phân Tích Chuyên Sâu Bản Gốc (Reference Audit)

Tài liệu này ghi nhận kết quả kiểm toán toàn diện từ **21 bài viết** trong chuỗi *Serverless Thực Chiến Trên AWS* (`kkloudtarus.net/blog/series/serverless-on-aws-event-driven`, trang 1 và trang 2) và **toàn bộ mã nguồn trên nhánh `main`** của repository `nghiadaulau/serverless-url-shortener-aws`.

---

## 1. Tổng Quan Chuỗi 21 Bài Viết (Trang 1 & Trang 2)

| Bài | Tiêu đề & URL | Nội dung kỹ thuật cốt lõi | Trạng thái ở cuối bài 21 (`main`) |
| --- | --- | --- | --- |
| **01** | `/blog/what-is-serverless-when-to-use` | Kiến trúc tổng thể hệ thống URL Shortener với Realtime Analytics trên AWS. | Giữ làm kim chỉ nam kiến trúc |
| **02** | `/blog/aws-sam-first-lambda-function` | Khởi tạo SAM, Lambda đầu tiên (`arm64`, `nodejs22.x`, `esbuild`). | Đã thay bằng 8 hàm nghiệp vụ thực tế |
| **03** | `/blog/how-lambda-runs-your-code` | Vòng đời Execution Environment (Init/Invoke), khởi tạo SDK ở cấp module để tái sử dụng kết nối warm. | Áp dụng trong `src/lib/db.ts`, `resolve-link.ts`, `ws.ts` |
| **04** | `/blog/api-gateway-http-vs-rest-api` | So sánh HTTP API v2 và REST API v1; định nghĩa `POST /links`, `GET /{code}`, hàm sinh Base62 7 ký tự. | Giữ nguyên trong `HttpApi` & `shortcode.ts` |
| **05** | `/blog/dynamodb-single-table-design` | Thiết kế bảng đơn `url-shortener` (`PK`, `SK`, `PAY_PER_REQUEST`), item collection `LINK#<code>` (`META` + `STAT#<date>`). | Giữ nguyên trong `template.yaml` |
| **06** | `/blog/dynamodb-gsi-and-sparse-index` | Thêm `GSI1` (`GSI1PK`, `GSI1SK`) tận dụng đặc tính Sparse Index (chỉ `META` và `CONN#` có khóa GSI1). | Giữ nguyên trong `template.yaml` |
| **07** | `/blog/dynamodb-conditional-writes-atomic-counter` | Chống trùng mã ngắn bằng `ConditionExpression: attribute_not_exists(PK)` (tối đa 5 lần thử). Bộ đếm trực tiếp ở `resolve-link` là bước trung gian trước bài 10. | Conditional write ở `create-link` giữ nguyên; bộ đếm ở `resolve-link` được thay bằng EventBridge ở bài 10 |
| **08** | `/blog/cognito-jwt-authorizer-http-api` | Thêm Cognito `UserPool`, `UserPoolClient` và `CognitoAuthorizer` (JWT) gắn riêng cho route cần bảo vệ; lấy `ownerId` từ `claims.sub`. | Giữ nguyên trong `template.yaml` & các handler |
| **09** | `/blog/multi-tenant-prevent-idor-dynamodb` | Thêm `GET /links` (query `GSI1` theo `USER#<sub>`) và `DELETE /links/{code}` (chặn IDOR bằng `ConditionExpression: attribute_exists(PK) AND ownerId = :me`). | Giữ nguyên trong `list-links.ts` & `delete-link.ts` |
| **10** | `/blog/eventbridge-publish-click-events` | Tách việc đếm click khỏi `resolve-link`: phát sự kiện `link.clicked` (`source: urlshortener`) lên custom bus `url-shortener-events` trước khi trả `301`. | Giữ nguyên trong `resolve-link.ts` |
| **11** | `/blog/lambda-sqs-idempotency-dlq-aggregator` | Đệm sự kiện qua `ClickQueue` + `ClickDLQ` (`maxReceiveCount: 3`), xử lý bằng `ClickAggregatorFunction` với `TransactWriteItems` (tăng `META.clicks`, tăng `STAT#<date>.count`, ghi marker `CLICK#<id>` có TTL 24h) và `ReportBatchItemFailures`. | Giữ nguyên trong `click-aggregator.ts` |
| **12** | `/blog/api-gateway-websocket-realtime-push` | Thêm `WebSocketApi` (`$connect`, `$disconnect`), lưu `CONN#<connId>` kèm `GSI1PK = WSCODE#<code>` (TTL 2h), đẩy `{ code, clicks }` qua `PostToConnection` và dọn kết nối chết khi gặp `410 GoneException`. | Giữ nguyên trong `ws-connect.ts`, `ws-disconnect.ts`, `ws.ts` |
| **13** | `/blog/step-functions-workflow-saga` | Thêm Step Functions `LinkModerationStateMachine` (`CheckSafety -> IsSafe -> Activate/Reject/MarkError`) cập nhật trực tiếp DynamoDB; giải thích thêm lý thuyết Saga. | State Machine độc lập trong `statemachine/moderation.asl.json` |
| **14** | `/blog/lambda-powertools-xray-observability` | Tích hợp Lambda Powertools (Logger, Tracer X-Ray, Metrics EMF `LinkResolved`/`LinkNotFound`) cho `resolve-link`. | Giữ nguyên trong `powertools.ts` & `resolve-link.ts` |
| **15** | `/blog/cloudwatch-alarms-dashboard-slo` | Thêm SNS `AlarmTopic`, 2 CloudWatch Alarms (`ResolveThrottleAlarm`, `ResolveErrorAlarm`) và CloudWatch Dashboard 3 widgets. | Giữ nguyên trong `template.yaml` |
| **16** | `/blog/lambda-cold-start-optimization` | Thử nghiệm tăng RAM 128 MB → 512 MB để giảm cold start, sau đó đưa về 128 MB để tiết kiệm chi phí; phân tích SnapStart và Provisioned Concurrency. | Cấu hình cuối cùng giữ `MemorySize: 128` |
| **17** | `/blog/serverless-security-iam-least-privilege-waf` | Siết IAM least-privilege riêng cho từng hàm, bật API Gateway throttling (`BurstLimit: 2`, `RateLimit: 5`), bàn luận lý thuyết SSM và WAF. | IAM least-privilege & throttling có trong `template.yaml` |
| **18** | `/blog/serverless-cicd-canary-rollback` | Thêm `.github/workflows/ci.yml` và cấu hình CodeDeploy `Canary10Percent5Minutes` kèm `AutoPublishAlias: live` + tự động rollback theo `ResolveErrorAlarm` cho `ResolveLinkFunction`. | Có trong `template.yaml` & `ci.yml` |
| **19** | `/blog/serverless-cost-breakdown` | Phân tích chi phí thực tế trên AWS Cost Explorer và các bẫy chi phí ngầm (NAT Gateway, Provisioned Concurrency, CloudWatch). | Tài liệu vận hành & kiểm soát chi phí |
| **20** | `/blog/serverless-load-test-k6` | Kịch bản k6 kiểm thử tải endpoint `GET /{code}` (40 VUs), quan sát cơ chế cắt tải 2 tầng (`429` ở API Gateway và `503` ở Lambda). | Script k6 nằm trong nội dung bài viết |
| **21** | `/blog/serverless-capstone-well-architected` | Đánh giá hệ thống theo 5 trụ cột AWS Well-Architected, hướng dẫn dọn dẹp và đề xuất hướng mở rộng. | Trạng thái hoàn chỉnh cuối series + `web/index.html` |

---

## 2. Phân Tích Chi Tiết Source Code Gốc (`nghiadaulau/serverless-url-shortener-aws`)

### 2.1 Cấu Trúc File Của Repository Gốc
Repository gốc trên nhánh `main` gồm 19 file:
- `template.yaml` (549 dòng): Định nghĩa toàn bộ tài nguyên SAM/CloudFormation.
- `src/lib/db.ts`: Khởi tạo `DynamoDBDocumentClient` bọc X-Ray, hằng `TABLE` (`TABLE_NAME ?? "url-shortener"`), hàm `linkKey(code)`.
- `src/lib/powertools.ts`: Khởi tạo `Logger`, `Tracer`, `Metrics` (`namespace: "UrlShortener"`, `serviceName: "url-shortener"`).
- `src/lib/shortcode.ts`: Hàm `generateShortCode(length = 7)` dùng `crypto.randomBytes` và `normalizeUrl(input)` chỉ nhận `http:` / `https:`.
- `src/lib/ws.ts`: Hàm `pushClickUpdate(code)` đọc `clicks` mới nhất, query `GSI1` theo `WSCODE#<code>`, gửi `{ code, clicks }` qua `PostToConnection` và xóa `CONN#<connId>` nếu gặp `GoneException` (HTTP 410).
- `src/handlers/create-link.ts`: Xử lý `POST /links`.
- `src/handlers/resolve-link.ts`: Xử lý `GET /{code}`.
- `src/handlers/list-links.ts`: Xử lý `GET /links`.
- `src/handlers/delete-link.ts`: Xử lý `DELETE /links/{code}`.
- `src/handlers/click-aggregator.ts`: Xử lý batch SQS từ `ClickQueue`.
- `src/handlers/ws-connect.ts`: Xử lý WebSocket `$connect`.
- `src/handlers/ws-disconnect.ts`: Xử lý WebSocket `$disconnect`.
- `src/handlers/moderate-check.ts`: Xử lý bước `CheckSafety` của Step Functions.
- `statemachine/moderation.asl.json`: Định nghĩa ASL cho `LinkModerationStateMachine`.
- `web/index.html`: Giao diện HTML/JS đơn giản gọi Cognito `InitiateAuth` (`USER_PASSWORD_AUTH`), gọi HTTP API và mở kết nối WebSocket theo từng mã link.
- `.github/workflows/ci.yml`: Chạy `npm ci`, `sam build`, `sam validate --lint`.

### 2.2 Chi Tiết 8 Hàm Lambda & Quyền IAM Least-Privilege Trong Bản Gốc

| Hàm Lambda | Trigger | Biến môi trường | Quyền IAM (`template.yaml`) |
| --- | --- | --- | --- |
| `CreateLinkFunction` | `HttpApi` `POST /links` (JWT Authorizer) | `TABLE_NAME` | `dynamodb:PutItem` trên `Table.Arn` |
| `ResolveLinkFunction` | `HttpApi` `GET /{code}` (Public, Alias `live` + Canary) | `TABLE_NAME`, `EVENT_BUS` | `dynamodb:GetItem` trên `Table.Arn`; `events:PutEvents` trên `EventBus.Arn` |
| `ListLinksFunction` | `HttpApi` `GET /links` (JWT Authorizer) | `TABLE_NAME` | `dynamodb:Query` trên `Table.Arn` và `${Table.Arn}/index/GSI1` |
| `DeleteLinkFunction` | `HttpApi` `DELETE /links/{code}` (JWT Authorizer) | `TABLE_NAME` | `dynamodb:DeleteItem` trên `Table.Arn` |
| `ClickAggregatorFunction` | `SQS` `ClickQueue` (`BatchSize: 10`, `ReportBatchItemFailures`) | `TABLE_NAME`, `WS_ENDPOINT` | `dynamodb:UpdateItem`, `PutItem`, `GetItem`, `DeleteItem`, `Query` trên `Table.Arn` + `GSI1`; `execute-api:ManageConnections` trên `WebSocketApi/*` |
| `WsConnectFunction` | `WebSocketApi` `$connect` | `TABLE_NAME` | `dynamodb:PutItem` trên `Table.Arn` |
| `WsDisconnectFunction` | `WebSocketApi` `$disconnect` | `TABLE_NAME` | `dynamodb:DeleteItem` trên `Table.Arn` |
| `ModerateCheckFunction` | Step Functions `CheckSafety` | *(Không có)* | Quyền mặc định ghi CloudWatch Logs & X-Ray |

---

## 3. Phân Biệt Tính Năng Đã Code Thực Tế Với Nội Dung Chỉ Thảo Luận Mở Rộng

| Hạng mục | Đã code trong Repo gốc? | Ghi chú xác minh thực tế |
| --- | --- | --- |
| DynamoDB Single-Table (`PK`/`SK`) + Sparse `GSI1` + `ttl` | **Đã code** | Bảng `url-shortener`, `PAY_PER_REQUEST` |
| Cognito User Pool + JWT Authorizer cho HTTP API | **Đã code** | Bảo vệ `POST /links`, `GET /links`, `DELETE /links/{code}`; `GET /{code}` mở công khai |
| Luồng sự kiện `EventBridge` → `SQS` + `DLQ` → `ClickAggregator` | **Đã code** | `TransactWriteItems` 3 thao tác + `ReportBatchItemFailures` |
| WebSocket `$connect`, `$disconnect` & `PostToConnection` | **Đã code** | Đẩy `{ code, clicks }` tới các kết nối đang theo dõi mã `code` |
| Step Functions `LinkModerationStateMachine` | **Đã code độc lập** | Được khai báo trong `template.yaml` và `statemachine/moderation.asl.json`, nhưng **không được tự động kích hoạt** khi tạo link và `resolve-link` **không chặn** theo trường `status` |
| Powertools (Log/Trace/Metrics) + CloudWatch Alarms + Dashboard | **Đã code** | Áp dụng cho `resolve-link` và `db.ts`; có 2 alarms + SNS topic + Dashboard 3 widgets |
| CodeDeploy Canary 10% 5 phút + Auto Rollback | **Đã code** | Khai báo trên `ResolveLinkFunction` với alias `live` và `ResolveErrorAlarm` |
| GitHub Actions OIDC Deploy tự động lên AWS | **Chưa code trong repo (chỉ giải thích ở bài 18)** | File `.github/workflows/ci.yml` gốc chỉ chạy `sam build` và `sam validate --lint`. Trong bản port này, ta sẽ triển khai đầy đủ cả CI và CD dùng GitHub OIDC + Terraform + CodeDeploy + S3/CloudFront theo yêu cầu Mục 11 |
| Kịch bản Load Test `k6` | **Nằm trong bài viết 20** | Chưa có file trong repo gốc; ta sẽ đóng gói vào `tests/load/redirect.k6.js` với chế độ mặc định tải nhỏ an toàn và chế độ stress riêng |
| AWS WAF / Provisioned Concurrency / Route 53 Custom Domain | **Chỉ giải thích mở rộng** | Tốn phí cố định theo giờ/tháng; giữ tắt mặc định đúng theo yêu cầu dự án |

---

## 4. Các Vấn Đề Kỹ Thuật Trong Bản Gốc: Cái Gì Sửa & Cái Gì Giữ Nguyên

### 4.1 Những Lỗi Kỹ Thuật Phải Sửa Trong Bản Go Để Đảm Bảo Chính Xác Dữ Liệu

1. **Phân biệt nguyên nhân `TransactionCanceledException` trong `ClickAggregator` (Bắt buộc theo Mục 7):**
   - **Vấn đề ở bản gốc:** `click-aggregator.ts` bắt mọi `TransactionCanceledException` và coi toàn bộ là sự kiện trùng (`duplicate, skip`). Tuy nhiên khi nhiều click xảy ra đồng thời trên cùng một mã link, DynamoDB có thể trả về `TransactionCanceledException` với `CancellationReasons[0].Code = "TransactionConflict"` hoặc `CancellationReasons[1].Code = "TransactionConflict"`. Bản gốc nuốt luôn lỗi này khiến sự kiện bị mất vĩnh viễn mà không được SQS retry.
   - **Cách sửa trong bản Go:** Kiểm tra mảng `CancellationReasons` của `*types.TransactionCanceledException`. Chỉ bỏ qua như duplicate khi `len(reasons) == 3 && reasons[2].Code != nil && *reasons[2].Code == "ConditionalCheckFailed"` (tức chỉ có điều kiện `attribute_not_exists(PK)` của `CLICK#<eventId>` bị vi phạm). Mọi lý do hủy khác (`TransactionConflict`, `ProvisionedThroughputExceeded`, v.v.) phải trả về lỗi để đưa `messageId` vào `batchItemFailures` cho SQS retry.

2. **Cô lập lỗi đẩy WebSocket khỏi kết quả xử lý Transaction (Bắt buộc theo Mục 7):**
   - **Vấn đề ở bản gốc:** Trong `click-aggregator.ts`, lệnh `await pushClickUpdate(code)` nằm chung trong khối `try` của record SQS. Nếu API Gateway Management API gặp lỗi mạng tạm thời, record bị đánh dấu thất bại và SQS gửi lại.
   - **Cách sửa trong bản Go:** Sau khi `TransactWriteItems` thành công, việc gọi `pushClickUpdate(ctx, code)` được bọc xử lý lỗi riêng: ghi structured warning log nếu việc đẩy WebSocket thất bại (trừ `410 GoneException` thì xóa connection), nhưng **không** đánh dấu message SQS thất bại và tuyệt đối không làm tăng bộ đếm lần hai.

3. **Cấu hình `PUBLIC_REDIRECT_BASE_URL` cho `shortUrl` (Bắt buộc theo Mục 5):**
   - **Vấn đề khi tách Frontend CloudFront và API Gateway:** Bản gốc ghép `https://${event.requestContext.domainName}/${code}`. Trong bản đích, để đảm bảo luôn tạo đúng `shortUrl` trỏ về public redirect base URL của API Gateway (không bị nhầm sang domain CloudFront của frontend), Lambda `create-link` ưu tiên đọc biến môi trường `PUBLIC_REDIRECT_BASE_URL` do Terraform truyền vào (fallback về `https://${event.requestContext.domainName}` nếu không cấu hình).

4. **Hỗ trợ cả Cognito Hosted UI (Authorization Code + PKCE) và User Password Auth:**
   - Bản gốc chỉ bật `ALLOW_USER_PASSWORD_AUTH` cho `web/index.html`. Trong Terraform mới, `aws_cognito_user_pool_client` được cấu hình đầy đủ OAuth2 Authorization Code + PKCE (`code` flow, Hosted UI domain, callback/logout URLs trỏ về CloudFront & localhost) theo đúng yêu cầu Mục 9, đồng thời giữ `ALLOW_USER_SRP_AUTH` / `ALLOW_USER_PASSWORD_AUTH` để phục vụ script kiểm thử tích hợp tự động.

### 4.2 Những Hành Vi & Giới Hạn Của Bản Gốc Được Giữ Nguyên (Có Ghi Chú Rõ)

1. **Step Functions Moderation (`LinkModerationStateMachine`) giữ nguyên là workflow độc lập:**
   - Giữ đúng định nghĩa `statemachine/moderation.asl.json` (`CheckSafety` → `IsSafe` → `Activate` / `Reject` / `MarkError`) cập nhật thuộc tính `status` (`active` / `rejected`) trên item `LINK#<code>` (`SK = META`).
   - Không tự ý gắn đồng bộ vào `POST /links` hay chặn `GET /{code}` để bảo toàn 100% hợp đồng API và độ trễ thấp của bản gốc; cung cấp tài liệu và kịch bản kích hoạt thực thi rõ ràng.
2. **Giới hạn bảo mật của WebSocket `$connect` gốc:**
   - Route `$connect?code=<code>` trong bản gốc không gắn authorizer và mỗi kết nối đăng ký theo một `code`. Ta giữ nguyên hành vi này để tương thích hoàn toàn với thiết kế bảng đơn (`PK = CONN#<connectionId>`, `GSI1PK = WSCODE#<code>`), đồng thời ghi rõ giới hạn bảo mật trong tài liệu kiến trúc.
3. **Thống kê theo ngày (`STAT#<YYYY-MM-DD>`):**
   - `ClickAggregator` vẫn ghi đầy đủ item `PK = LINK#<code>, SK = STAT#<YYYY-MM-DD>` trong cùng transaction như bản gốc. Dashboard Next.js chỉ sử dụng các API gốc (`POST /links`, `GET /links`, `DELETE /links/{code}`) và luồng WebSocket `{ code, clicks }` mà không tự ý bịa thêm endpoint mới.

---

## 5. Bảng Mapping Chi Tiết: Bản Gốc → Bản Port (Go + Terraform + Next.js)

| Thành phần gốc | Vị trí trong Repo gốc | Thành phần mới (Go / Terraform / Next.js) | Vị trí trong Monorepo mới |
| --- | --- | --- | --- |
| SAM Template | `template.yaml` | Hệ thống Terraform Modules & Environment | `infrastructure/terraform/modules/*` & `infrastructure/terraform/environments/dev/*` |
| Terraform Remote State | *(SAM tự tạo S3 bucket)* | Bootstrap S3 State Bucket (Versioning + Encryption + Lockfile) | `infrastructure/terraform/bootstrap/*` |
| Shared DB Client | `src/lib/db.ts` | Go DynamoDB Repository & Client khởi tạo kèm X-Ray | `backend/internal/repository/dynamodb.go` |
| Base62 & URL Validator | `src/lib/shortcode.ts` | Go Shortcode Generator (`crypto/rand`) & URL Validator | `backend/internal/domain/shortcode.go` |
| Powertools (Log/Trace/EMF) | `src/lib/powertools.ts` | Go Observability Middleware (`log/slog` JSON, CloudWatch EMF, X-Ray) | `backend/internal/middleware/observability.go` |
| WebSocket Pusher | `src/lib/ws.ts` | Go API Gateway Management API Pusher (kèm xóa `410 Gone`) | `backend/internal/events/ws_pusher.go` |
| `CreateLinkFunction` | `src/handlers/create-link.ts` | Go Lambda `create-link` (`provided.al2023`, `arm64`) | `backend/cmd/create-link/main.go` |
| `ResolveLinkFunction` | `src/handlers/resolve-link.ts` | Go Lambda `resolve-link` (`provided.al2023`, `arm64`, alias `live`) | `backend/cmd/resolve-link/main.go` |
| `ListLinksFunction` | `src/handlers/list-links.ts` | Go Lambda `list-links` (`provided.al2023`, `arm64`) | `backend/cmd/list-links/main.go` |
| `DeleteLinkFunction` | `src/handlers/delete-link.ts` | Go Lambda `delete-link` (`provided.al2023`, `arm64`) | `backend/cmd/delete-link/main.go` |
| `ClickAggregatorFunction` | `src/handlers/click-aggregator.ts` | Go Lambda `click-aggregator` (`provided.al2023`, `arm64`) | `backend/cmd/click-aggregator/main.go` |
| `WsConnectFunction` | `src/handlers/ws-connect.ts` | Go Lambda `ws-connect` (`provided.al2023`, `arm64`) | `backend/cmd/ws-connect/main.go` |
| `WsDisconnectFunction` | `src/handlers/ws-disconnect.ts` | Go Lambda `ws-disconnect` (`provided.al2023`, `arm64`) | `backend/cmd/ws-disconnect/main.go` |
| `ModerateCheckFunction` | `src/handlers/moderate-check.ts` | Go Lambda `moderate-check` (`provided.al2023`, `arm64`) | `backend/cmd/moderate-check/main.go` |
| Step Functions ASL | `statemachine/moderation.asl.json` | Giữ nguyên cấu trúc ASL chuẩn, truyền ARN/TableName qua Terraform `templatefile` | `statemachine/moderation.asl.json` |
| Vanilla HTML Dashboard | `web/index.html` | Next.js App Router (`output: "export"`) + TypeScript + Tailwind CSS trên S3 + CloudFront OAC | `frontend/src/*` |
| CI Workflow | `.github/workflows/ci.yml` | GitHub Actions CI + CD (OIDC, Go build/test, Next.js export, Terraform, CodeDeploy Canary, S3 sync + CloudFront invalidation) | `.github/workflows/ci.yml` & `.github/workflows/deploy.yml` |
| Load Test Script | *(Trong bài viết 20)* | Kịch bản k6 với chế độ smoke mặc định và stress test theo biến môi trường | `tests/load/redirect.k6.js` |
