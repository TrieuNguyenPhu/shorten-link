# Phase 0 — Phân Tích Chuyên Sâu Bản Gốc (Reference Audit)

Nguồn được đọc ngày **2026-10-26**, tài liệu hoàn thiện ngày **2026-10-27**. Phạm vi gồm toàn bộ thân bài (code, output và kết luận) của **21 bài** trên [trang 1](https://kkloudtarus.net/blog/series/serverless-on-aws-event-driven) và [trang 2](https://kkloudtarus.net/blog/series/serverless-on-aws-event-driven?page=2), cùng **21 tracked files** tại commit [`a5ffed9780d0447fc75a42def3693732bdc87907`](https://github.com/nghiadaulau/serverless-url-shortener-aws/tree/a5ffed9780d0447fc75a42def3693732bdc87907) trên `main` khi đọc.

| Người kiểm chứng trong nhóm | Phạm vi đã đọc |
| --- | --- |
| Agent đọc nhóm 1–7 | Đủ thân bài 1–7, các bước chuyển từ demo sang persistence |
| Agent đọc nhóm 8–14 | Đủ thân bài 8–14; đã đọc được pagination trang 2 sau lỗi mở trang tạm thời |
| Agent đọc nhóm 15–21 | Đủ thân bài 15–21, kết quả demo, giới hạn benchmark và mở rộng |
| Controller | Template, 8 handlers, 4 thư viện, ASL, web, CI và metadata repository |

Đây là audit tĩnh của nguồn và bằng chứng bài viết, không phải lần deploy/chạy AWS của nhóm. Không có nguồn còn thiếu; không suy diễn demo của tác giả thành kết quả kiểm thử bản Go. Việc dọn AWS cũ là xác nhận của người dùng, không được kiểm kê lại trong phiên hoàn thiện tài liệu.

---

## 1. Tổng Quan Chuỗi 21 Bài Viết (Trang 1 & Trang 2)

| Bài | Tiêu đề & URL | Nội dung kỹ thuật cốt lõi | Trạng thái ở cuối bài 21 (`main`) |
| --- | --- | --- | --- |
| **01** | [Tổng quan serverless](https://kkloudtarus.net/blog/what-is-serverless-when-to-use) | Sơ đồ mục tiêu URL Shortener. | WAF/luồng moderation trên sơ đồ không chứng minh được nối trong code; WAF chưa dựng. |
| **02** | [Lambda đầu tiên với SAM](https://kkloudtarus.net/blog/aws-sam-first-lambda-function) | Lambda arm64, nodejs22.x, esbuild. | Hello Function URL là bước tạm, được thay bằng 8 Lambda nghiệp vụ. |
| **03** | [Vòng đời Lambda](https://kkloudtarus.net/blog/how-lambda-runs-your-code) | Init/Invoke, tái sử dụng SDK client khi warm. | Demo CPU/cold start Node không phải baseline Go. |
| **04** | [HTTP API và REST API](https://kkloudtarus.net/blog/api-gateway-http-vs-rest-api) | POST /links, GET /{code}, Base62 7 ký tự. | Create chưa persist, resolve URL giả; thay ở bài 7. Cache 301 có thể bỏ qua backend ở lần mở sau. |
| **05** | [DynamoDB single-table](https://kkloudtarus.net/blog/dynamodb-single-table-design) | PK/SK, PAY_PER_REQUEST, META/STAT item collection. | Daily collection chỉ CLI demo; không có endpoint daily analytics. |
| **06** | [GSI và sparse index](https://kkloudtarus.net/blog/dynamodb-gsi-and-sparse-index) | GSI1PK/GSI1SK, META sparse index. | CONN thêm ở bài 12; không gán ngược cho bài 6. |
| **07** | [Conditional writes và counter](https://kkloudtarus.net/blog/dynamodb-conditional-writes-atomic-counter) | attribute_not_exists(PK), tối đa 5 lần ghi. | Owner user-001 và DynamoDBCrudPolicy là tạm; chống collision không là idempotency POST. Counter trực tiếp thay ở bài 10. |
| **08** | [Cognito JWT](https://kkloudtarus.net/blog/cognito-jwt-authorizer-http-api) | User Pool, App Client, JWT authorizer, claims.sub. | Có trong main cho 3 route cần bảo vệ; resolve public. |
| **09** | [Multi-tenant và IDOR](https://kkloudtarus.net/blog/multi-tenant-prevent-idor-dynamodb) | Query USER#sub, delete condition tồn tại và ownerId=:me. | Có trong main; không lấy owner từ client. |
| **10** | [Publish click qua EventBridge](https://kkloudtarus.net/blog/eventbridge-publish-click-events) | source=urlshortener, detail-type=link.clicked, bus url-shortener-events. | ClickLogger nhận EventBridge trực tiếp là tạm, thay bằng SQS/aggregator ở bài 11. |
| **11** | [SQS, idempotency và DLQ](https://kkloudtarus.net/blog/lambda-sqs-idempotency-dlq-aggregator) | Transaction META/STAT/CLICK, ReportBatchItemFailures. | Marker+24h, batch10/visibility30/maxReceive3/DLQ14d; dedupe hữu hạn, không exactly-once vô hạn. |
| **12** | [WebSocket realtime](https://kkloudtarus.net/blog/api-gateway-websocket-realtime-push) | CONN, WSCODE GSI, TTL2h, PostToConnection và cleanup410. | Connect công khai: ai biết code đều theo dõi count được; TTL không là authorization. |
| **13** | [Step Functions và Saga](https://kkloudtarus.net/blog/step-functions-workflow-saga) | CheckSafety → IsSafe → Activate/Reject; Catch → MarkError. | Độc lập/manual: không pending/StartExecution, không chặn redirect; MarkError là Fail, không ghi status=error. |
| **14** | [Powertools và X-Ray](https://kkloudtarus.net/blog/lambda-powertools-xray-observability) | Logger/Tracer/EMF LinkResolved/LinkNotFound. | Chỉ resolve có đủ middleware; tracing Active mọi Lambda không đồng nghĩa mọi handler có JSON/EMF. |
| **15** | [Alarms, dashboard và SLO](https://kkloudtarus.net/blog/cloudwatch-alarms-dashboard-slo) | SNS, ResolveThrottleAlarm, ResolveErrorAlarm, dashboard3widgets. | SNS chưa subscription; alarms Sum>=1/60s/1period/notBreaching. SLO99.9%/p99<200ms là mục tiêu ví dụ chưa enforcement. |
| **16** | [Tối ưu cold start](https://kkloudtarus.net/blog/lambda-cold-start-optimization) | Thử RAM128→512, phân tích SnapStart/Provisioned Concurrency. | Final memory128/timeout5/arm64. Benchmark Node và quota tài khoản10 không phải baseline Go. |
| **17** | [IAM, throttling và WAF](https://kkloudtarus.net/blog/serverless-security-iam-least-privilege-waf) | Thu hẹp IAM theo hàm, API throttling. | Final rate5/burst2/CORS*. WAF/SSM chỉ thảo luận. |
| **18** | [CI/CD và canary](https://kkloudtarus.net/blog/serverless-cicd-canary-rollback) | live alias, canary10%5min, ResolveErrorAlarm. | CI build/validate only, OIDC chưa có. Demo success, chưa chứng minh rollback bằng fault injection. |
| **19** | [Phân tích chi phí](https://kkloudtarus.net/blog/serverless-cost-breakdown) | Cost Explorer, bẫy NAT/Provisioned Concurrency/Logs. | Hóa đơn lịch sử tác giả không bảo đảm giá hiện tại hay chi phí bằng0 cho tài khoản khác. |
| **20** | [Kiểm thử k6](https://kkloudtarus.net/blog/serverless-load-test-k6) | 40VU, 429/503, redirects:0. | Script chỉ trong bài; nhiều failed, ~554RPS không phải successful throughput. |
| **21** | [Capstone Well-Architected](https://kkloudtarus.net/blog/serverless-capstone-well-architected) | Tổng kết, cleanup, hướng mở rộng. | Dashboard được nhắc như mở rộng nhưng main có HTML login/create/list/delete/WS; chưa hosting S3/CloudFront. |

---

## 2. Phân Tích Chi Tiết Source Code Gốc (`nghiadaulau/serverless-url-shortener-aws`)

### 2.1 Cấu Trúc File Của Repository Gốc
Repository tại SHA nêu trên gồm **21 tracked files** (đếm bằng `git ls-files`):
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
- `.gitignore`, `README.md`, `package.json`, `package-lock.json`, `tsconfig.json`: 5 file metadata/cấu hình còn lại. Lockfile version 3 có 109 packages; không suy diễn đó là baseline dependency Go/Next.js.

### 2.2 Chi Tiết 8 Hàm Lambda & Quyền IAM Trong Bản Gốc

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

SAM còn sinh quyền poll SQS (`ReceiveMessage`, `DeleteMessage`, `GetQueueAttributes`) cho consumer, quyền Logs/X-Ray và role Step Functions. Terraform phải khai báo tường minh, scope theo resource; bảng trên không chứng nhận mọi policy nguồn đã tối thiểu tuyệt đối.

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
| GitHub Actions OIDC Deploy tự động lên AWS | **Chưa code trong repo** | CI gốc build/validate; bản port bổ sung bootstrap OIDC ở Task 2.2 và CD ở Task 6.2 |
| Kịch bản Load Test `k6` | **Nằm trong bài viết 20** | Chưa có file trong repo gốc; ta sẽ đóng gói vào `tests/load/redirect.k6.js` với chế độ mặc định tải nhỏ an toàn và chế độ stress riêng |
| AWS WAF / Provisioned Concurrency / Route 53 Custom Domain | **Chỉ giải thích mở rộng** | Tốn phí cố định theo giờ/tháng; giữ tắt mặc định đúng theo yêu cầu dự án |

---

## 4. Các Vấn Đề Kỹ Thuật Trong Bản Gốc: Cái Gì Sửa & Cái Gì Giữ Nguyên

### 4.1 Những Lỗi Kỹ Thuật Phải Sửa Trong Bản Go Để Đảm Bảo Chính Xác Dữ Liệu

Hợp đồng lỗi đầy đủ nằm tại [đặc tả kiến trúc](architecture-and-contracts.md), mục 3.4, 5.2 và 5.4. Các delta có chủ đích:

1. **Phân biệt nguyên nhân `TransactionCanceledException` trong `ClickAggregator`:**
   - **Vấn đề ở bản gốc:** `click-aggregator.ts` bắt mọi `TransactionCanceledException` và coi toàn bộ là sự kiện trùng (`duplicate, skip`). Tuy nhiên khi nhiều click xảy ra đồng thời trên cùng một mã link, DynamoDB có thể trả về `TransactionCanceledException` với `CancellationReasons[0].Code = "TransactionConflict"` hoặc `CancellationReasons[1].Code = "TransactionConflict"`. Bản gốc nuốt luôn lỗi này khiến sự kiện bị mất vĩnh viễn mà không được SQS retry.
   - **Cách sửa trong bản Go:** Kiểm tra đủ cả ba lý do với bounds/nil checks; duplicate chỉ khi marker conditional failure và hai lý do khác `None`. Operational/mixed/unknown errors phải retry. META update thêm condition tồn tại; ack late-deleted chỉ sau consistent read xác nhận META mất, không tạo STAT/marker hoặc push. Không đổi schema để giải quyết code reuse.

2. **Cô lập lỗi đẩy WebSocket khỏi kết quả transaction:**
   - **Bản gốc:** `ws.ts` nuốt lỗi `PostToConnection` non-410; lỗi DynamoDB Get/Query/cleanup mới propagate qua `pushClickUpdate` ra khối catch SQS. Vì vậy không đúng khi nói mọi lỗi mạng push đều làm SQS retry.
   - **Bản Go:** Cô lập toàn bộ notifier sau commit, log lỗi; record đã commit vẫn thành công, push best-effort. Đọc META consistent, paginate mọi connection và list user links qua `LastEvaluatedKey`; GSI vẫn eventual consistency. UI reconnect rồi nạp lại GET links, hợp nhất count bằng max.

3. **Cấu hình `PUBLIC_REDIRECT_BASE_URL` cho `shortUrl`:**
   - **Vấn đề khi tách Frontend CloudFront và API Gateway:** Bản gốc ghép `https://${event.requestContext.domainName}/${code}`. Trong bản đích, để đảm bảo luôn tạo đúng `shortUrl` trỏ về public redirect base URL của API Gateway (không bị nhầm sang domain CloudFront của frontend), Lambda `create-link` ưu tiên đọc biến môi trường `PUBLIC_REDIRECT_BASE_URL` do Terraform truyền vào (fallback về `https://${event.requestContext.domainName}` nếu không cấu hình).

4. **Google-only qua Cognito User Pool + Authorization Code/PKCE:** Đây là thay đổi theo yêu cầu người dùng sau audit. Bản gốc dùng password auth; bản port chỉ hỗ trợ Google social IdP, không cung cấp form đăng nhập/đăng ký hay password flow kể cả E2E. Cognito cấp token cho ứng dụng; API dùng Cognito access token. Không mang password demo hoặc nội suy `innerHTML` từ nguồn sang React.

5. **Kiểm tra publish:** `resolve-link.ts` không kiểm `FailedEntryCount`/`Entries.ErrorCode`. Go kiểm từng entry, retry có giới hạn và chỉ trả 301 khi có EventId; lỗi phát metric và Lambda error/API5xx. Timeout publish bất định và bus thiếu có thể làm mất/đếm lại event; xem hợp đồng 3.4.

### 4.2 Những Hành Vi & Giới Hạn Của Bản Gốc Được Giữ Nguyên (Có Ghi Chú Rõ)

1. **Step Functions Moderation (`LinkModerationStateMachine`) giữ nguyên là workflow độc lập:**
   - Giữ đúng định nghĩa `statemachine/moderation.asl.json` (`CheckSafety` → `IsSafe` → `Activate` / `Reject` / `MarkError`) cập nhật thuộc tính `status` (`active` / `rejected`) trên item `LINK#<code>` (`SK = META`).
   - Không tự ý gắn đồng bộ vào `POST /links` hay chặn `GET /{code}` để bảo toàn 100% hợp đồng API và độ trễ thấp của bản gốc; cung cấp tài liệu và kịch bản kích hoạt thực thi rõ ràng.
   - Substring blocklist `malware/phishing/evil`; Retry `Lambda.ServiceException`, `Lambda.TooManyRequestsException`, `States.TaskFailed` (1s/3/2), Catch `States.ALL`. `MarkError` là Fail, không ghi status=error. Giữ ASL Activate/Reject có thể upsert META thiếu dữ liệu: runbook tạo item hợp lệ trước và không chạy cùng xóa link.
2. **Giới hạn bảo mật của WebSocket `$connect` gốc:**
   - Route `$connect?code=<code>` trong bản gốc không gắn authorizer và mỗi kết nối đăng ký theo một `code`. Ta giữ nguyên hành vi này để tương thích hoàn toàn với thiết kế bảng đơn (`PK = CONN#<connectionId>`, `GSI1PK = WSCODE#<code>`), đồng thời ghi rõ giới hạn bảo mật trong tài liệu kiến trúc.
   - Ai biết code đều có thể theo dõi count; connection TTL không là authorization, push không có durable delivery.
3. **Thống kê theo ngày (`STAT#<YYYY-MM-DD>`):**
   - `ClickAggregator` vẫn ghi đầy đủ item `PK = LINK#<code>, SK = STAT#<YYYY-MM-DD>` trong cùng transaction như bản gốc. Dashboard Next.js chỉ sử dụng các API gốc (`POST /links`, `GET /links`, `DELETE /links/{code}`) và luồng WebSocket `{ code, clicks }` mà không tự ý bịa thêm endpoint mới.
4. **Giới hạn count:** giữ marker TTL+24h, không bỏ qua marker còn tồn tại chỉ vì TTL đã qua. Sau khi marker mất, replay có thể đếm lại; DLQ14d yêu cầu đối soát tuổi/cause trước redrive. Cache301, EventBridge id mới khi republish và code reuse ngăn việc hứa unique visits hoặc exactly-once vô hạn.

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
| Vanilla HTML Dashboard | `web/index.html` | Next.js App Router (`output: "export"`) + TypeScript + Tailwind CSS, Hand-Drawn / Doodle & Sketch UI trên S3 + CloudFront OAC | `frontend/src/*` |
| CI Workflow | `.github/workflows/ci.yml` | GitHub Actions CI + CD (OIDC, Go build/test, Next.js export, Terraform, CodeDeploy Canary, S3 sync + CloudFront invalidation) | `.github/workflows/ci.yml` & `.github/workflows/deploy.yml` |
| Load Test Script | *(Trong bài viết 20)* | Kịch bản k6 với chế độ smoke mặc định và stress test theo biến môi trường | `tests/load/redirect.k6.js` |

## 6. Bằng Chứng Skills Và Giới Hạn Xác Minh

Kiểm tra ngày 2026-10-26: **314** thư mục cấp đầu có `SKILL.md`, **376** file `SKILL.md` đệ quy (có trùng). `.agents/skills.json` và `~/.gemini/config/skills.json` cùng trỏ `~/.agents/skills`; không cài lại trong phiên này.

| Bộ khai báo | File đại diện đã thấy dưới `~/.agents/skills/` |
| --- | --- |
| obra/superpowers | `using-superpowers/SKILL.md` |
| DietrichGebert/ponytail | `ponytail/SKILL.md` |
| OthmanAdi/planning-with-files | `planning-with-files/SKILL.md` |
| JuliusBrussee/caveman | `caveman/SKILL.md` |
| trailofbits/skills | `differential-review/SKILL.md` |
| vercel-labs/agent-skills | `react-best-practices/SKILL.md` |
| anthropics/skills | `frontend-design/SKILL.md` |
| open-gsd/gsd-core | `gsd-help/SKILL.md` |
| Egonex-AI/Understand-Anything | `understand/SKILL.md` |
| alibaba/open-code-review | `open-code-review/SKILL.md` |
| nextlevelbuilder/ui-ux-pro-max-skill | `ui-ux-pro-max/SKILL.md` |
| Leonxlnx/taste-skill | `taste-skill/SKILL.md` |
| shanraisshan/claude-code-best-practice | `vibe-to-agentic-framework/SKILL.md` |

Hiện diện file và cấu hình không chứng minh mọi host tự động nạp skill, cũng không xác thực độc lập provenance/phiên bản từng bộ. Phase 0 hoàn thành ở mức audited specification, chưa có app hoặc AWS E2E được chạy.
