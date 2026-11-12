# Kế hoạch tái cấu trúc Shrnk theo SRS trên AWS serverless

> **Đã được thay thế ngày 12/11/2026.** Dùng [kế hoạch Go/Gin Lambda dưới $20](2026-11-12-go-gin-budget-plan.md) và [kiến trúc hiện hành](../specs/2026-11-12-go-gin-budget-architecture.md). Nội dung bên dưới chỉ lưu phương án trước, không dùng để thực thi.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax for tracking. This document is a preparation plan; creating it does not authorize implementation or deployment.

**Goal:** Đưa các nghiệp vụ FR-01 đến FR-06 của SRS vào Shrnk, giữ AWS serverless, tài khoản và link hiện có, với từng đợt chuyển đổi có kiểm chứng và rollback tương thích dữ liệu.

**Architecture:** Tái cấu trúc theo module nghiệp vụ trong Go, giữ các Lambda entrypoint và Next.js static frontend. Mở rộng schema theo thứ tự reader tương thích → writer mới → backfill → bỏ nhánh cũ. Analytics dùng nhánh đếm live riêng và nhánh raw/enrichment/rollup riêng.

**Tech Stack:** Go 1.26.10 hiện tại, Next.js 16/React 19, DynamoDB, Cognito, API Gateway HTTP/WebSocket, EventBridge/SQS, Step Functions, CloudFront/S3, Terraform. Đề xuất bổ sung Firehose, Athena/Glue, Scheduler khi triển khai pha sở hữu chúng.

**Spec:** [Thiết kế và đối chiếu SRS](../specs/2026-11-12-serverless-srs-refactor-design.md). Đọc cả hai tài liệu trước khi thực hiện; đây không phải cam kết đáp ứng nguyên xi stack/NFR của SRS container.

## Global Constraints

- Người dùng đã chọn: **Giữ AWS serverless, áp dụng nghiệp vụ SRS**. Chưa có yêu cầu sửa code hoặc deploy trong lần lập kế hoạch này.
- Giữ các thay đổi chưa commit có sẵn; không reset/stash/commit chung thay đổi của người dùng. Chỉ tạo branch/worktree cô lập lúc bắt đầu thực thi và đưa đúng baseline đang được duyệt sang đó.
- Không thay Cognito pool/sub, DynamoDB key `LINK#<code>/META`, domain hoặc mã link đã phát hành để tiện tái cấu trúc.
- Giữ Base62 7 ký tự, alias hiện tại 3–32, expiryDays 1–365 hoặc vĩnh viễn, URL policy, moderation và 302/no-store; thêm contract mới có chủ đích.
- Access token **15 phút**, refresh đề xuất **7 ngày** theo SRS; raw retention **90 ngày**; aggregate_window **10 phút**; daily rollup **00:05 UTC**; anomaly **15 phút**, baseline **7 ngày**.
- P95 **<10ms**, p99 **<25ms** tại **5.000 RPS**; **8.000 RPS** trên **2 vCPU/512MB**; cache hit **≥90%** và cache-hit **<5ms** là số gốc cần đối chiếu feasibility, không được báo đạt khi chưa đo.
- Không dùng goroutine sau khi Lambda return hoặc buffer RAM làm nơi lưu telemetry bền vững. Không có hai writer cùng cộng một lifetime counter.
- Link nghiệp vụ không bị xóa bởi TTL; soft delete giữ tombstone và analytics. Sau schema cutover chỉ rollback về runtime vCompat trở lên.
- Giữ tên và state Terraform trong đợt đầu; mọi tài nguyên mới gắn với một deliverable, có plan và chi phí được review trước apply.

## Review Focus

1. Link cũ, alias dài, thời điểm hết hạn và TTL chưa dọn: resolve/ownership phải đúng cả trước và sau backfill; kiểm ở P1/P3/P7.
2. JWT còn hạn nhưng account/role đã thay đổi, CSRF hoặc đoán jobId: backend phải từ chối thao tác và file export trái quyền; kiểm ở P2/P6.
3. Event v1/v2 giao trùng, đến muộn, replay hoặc consumer chết giữa chừng: không tăng click hai lần, không mất nhánh raw vì nhánh count đã thành công; kiểm ở P4/P5.
4. Scheduled/manual/backfill chồng cửa sổ, daily unique và retention: kết quả tái lập, không cộng unique từng giờ và không đọc dữ liệu đã hết retention; kiểm ở P5.
5. Owner cố gỡ khóa admin, quota đua nhau, filter thưa và client IP giả: không vượt quyền, không vượt quota, không treo pagination và không dùng IP tùy ý; kiểm ở P3/P4/P6.

## 1. Thứ tự và điểm dừng review

| Pha | Deliverable có thể nghiệm thu riêng | Phụ thuộc | Công sức dự trù |
|---|---|---|---|
| P0 | Baseline tái lập, phụ lục SRS áp dụng, access patterns và tiêu chí đo | Review kế hoạch | 2–3 ngày kỹ thuật |
| P1 | Module boundaries mới, giữ nguyên hành vi và resource names | P0 | 3–4 ngày |
| P2 | Profile/RBAC, đăng ký Cognito và session refresh | P1, chốt phương án auth | 4–7 ngày |
| P3 | vCompat, lifecycle, filters, quota và migration metadata | P1, policy P2 | 5–8 ngày |
| P4 | Event v2, raw/enrichment và realtime có giới hạn fan-out | P3 | 5–8 ngày |
| P5 | Rollup, job registry, backfill, compaction, anomaly | P4 | 5–8 ngày |
| P6 | Analytics/admin/ops UI, export có RBAC | P2–P5 | 5–7 ngày |
| P7 | NFR, release/rollback drill, staging và canary | Các pha trước | 3–5 ngày |

Ước lượng thô cho một người hiểu repo: khoảng **32–50 ngày kỹ thuật**, chưa bao gồm chờ duyệt, đo tải lớn, xử lý phát hiện mới hoặc vận hành production. Không dùng làm lịch cam kết. MVP nên kết thúc ở P3 với lifecycle/phân quyền đúng; P4–P6 là phần mở rộng analytics và vận hành. Mỗi pha chia PR nhỏ theo deliverable dưới đây, không gộp thành một PR toàn repo.

Trước code mỗi module mới, chốt spec/ADR và test contract của module đó từ thiết kế này. Những điểm sản phẩm chưa xác nhận chỉ chặn pha phụ thuộc; không chặn việc khảo sát, chốt baseline và chuẩn bị test fixtures.

## 2. File map và nguyên tắc chuyển đổi

| Hiện tại | Đích dự kiến | Trách nhiệm |
|---|---|---|
| `backend/internal/domain/{models,shortcode}.go` | `internal/shortener/{model,policy}.go`, `analytics/event.go`, `realtime/model.go` | Tách model nghiệp vụ, DTO storage/HTTP nằm trong adapter |
| `backend/internal/service/link_service.go` | `internal/shortener/{ports,service}.go` | Link use cases; bỏ import repository cụ thể và logic ticket khỏi listing |
| `backend/internal/repository/{dynamodb,cursor,aggregator,ws}.go` | `internal/adapters/dynamodb/{links,cursor,counter,connections}.go` | Giữ key/schema trước, chuyển package trước |
| `backend/internal/service/aggregator_service.go` | `analytics/{consumer,counter}.go`, `transport/queue/click.go` | Domain processing tách khỏi SQS event envelope |
| `backend/internal/{events,httpapi,wsapi}` | `adapters/{eventbridge,cognito,firehose,athena}`, `transport/{httpapi,wsapi}` | Boundary theo giao thức/dịch vụ |
| `backend/internal/{config,middleware}` | `platform/{config,telemetry}` | Config validation, logging và metrics dùng chung |
| `frontend/src/lib/{api,auth}.ts` | API/session client có cấu trúc theo feature | Giữ wrapper trong thời gian chuyển caller, không rename hàng loạt trước khi có nhu cầu |
| `frontend/src/features/links/*` | Giữ links; thêm `features/{auth,analytics,admin,operations}` | UI theo nghiệp vụ, pagination thực và role-aware navigation |
| README/types/tests rải rác | `contracts/openapi.yaml`, `contracts/events/*.schema.json` | Một hợp đồng độc lập, fixtures/test chống drift |
| Terraform và CI hiện có | Giữ root/module; thêm `analytics_storage`, `analytics_jobs` khi cần | Không tạo lại state hoặc bỏ gate đang bảo vệ hệ thống |

Các đường dẫn dưới `internal/` trong bảng là tương đối với `backend/`. Đường dẫn tạo mới ở phần task là kế hoạch, chưa phải file đã có. Có thể đổi tên file nhỏ khi implementation review, nhưng không được đổi public contract, ownership hoặc data key ngoài ADR.

## 3. Các task triển khai

### P0 — Chuẩn bị baseline và phạm vi áp dụng

**Files:** tạo `docs/srs/serverless-adaptation.md`, `docs/refactor/baseline.md`, `docs/refactor/access-patterns.md`, `docs/refactor/release-matrix.md`; tham chiếu SRS gốc, không đưa secret/state dump vào Git.

**Interfaces:** nhận working tree thực tế và SRS; tạo baseline có commit + hash diff, danh sách invariant, ma trận yêu cầu/ngoại lệ và quyết định đã duyệt cho các pha sau.

- [x] Đọc SRS, kiểm tra code/infra/frontend, ghi matrix FR/NFR trong thiết kế.
- [x] Chạy Go race tests/vet và 39 frontend tests làm điểm tham chiếu ban đầu.
- [ ] Chốt các điều chỉnh SRS ở mục 5 thiết kế, đặc biệt auth native, RBAC, 302, cursor, raw retention, budget và mục tiêu latency AWS.
- [ ] Chụp baseline thay đổi đang có theo cách không mất nội dung; tách worktree lúc thực thi. Ghi nhận các file audit/plan chưa track, không bỏ chúng khi lấy baseline.
- [ ] Chạy fresh build và các gate hiện có: `make check`; integration: `DYNAMODB_ENDPOINT=http://127.0.0.1:8000 make test-integration`. Ghi rõ dependency/tool thiếu hoặc test fail, không đổi code ngoài phạm vi chỉ để làm baseline xanh.
- [ ] Khi có quyền đọc môi trường, lập inventory alias đang dùng, Lambda aliases, route/callback, DynamoDB/PITR, TTL, trạng thái queue, Terraform state addresses và deployment boundary; không coi README/audit cũ là cloud state hiện tại.
- [ ] Lưu latency baseline bằng smoke phù hợp môi trường; thiết kế bài đo 100/500/1.000/5.000 RPS riêng, có quota/cost envelope trước khi chạy. Phân biệt peak và sustained, public latency và Lambda duration.

**Gate:** baseline tái lập, mọi regression đang có được phân loại, chính sách tương thích và tài nguyên được ghi lại. P0 không bao gồm deploy hay dọn các thay đổi cũ.

### P1 — Tách module mà không đổi hành vi

**Files:** tạo `backend/internal/shortener/{model,ports,policy,service}.go`, `analytics/{event,consumer}.go`, `realtime/{model,service}.go`, `adapters/dynamodb/*`, `adapters/eventbridge/*`, `transport/{httpapi,wsapi,queue}/*`, `platform/{config,telemetry}/*`; sửa `backend/cmd/*/main.go`, `backend/tools/local-http/*`; di chuyển test cùng package.

**Interfaces:** `shortener.LinkStore` giữ bốn thao tác `CreateLink(ctx, Link) error`, `GetLink(ctx, code) (Link, error)`, `ListUserLinks(ctx, owner, limit, cursor) (LinkPage, error)`, `DeleteUserLink(ctx, code, owner) error` trong pha tương thích. Use case nhận `context.Context` và type nghiệp vụ; queue adapter chuyển `SQSEvent` thành `analytics.Click`, không truyền AWS envelope vào service.

- [ ] Bổ sung characterization tests cho fixture hiện có: create/alias collision/moderation, 302/no-store, expiry boundary, owner-scoped cursor/delete, v1 click dedupe, ticket một lần. Chỉ thêm ca mà baseline chưa bảo vệ, không viết lại test tương đương.
- [ ] Chạy các ca mới trước thay đổi để xác nhận chúng bảo vệ hành vi hiện tại; ca này không bắt buộc đỏ vì đây là characterization, không phải hành vi mới.
- [ ] Di chuyển domain và ports trước, adapter sau, entrypoint cuối; giữ wrapper tạm để repo compile giữa các commit. Chuyển lỗi `ErrNotFound/ErrCollision` về biên nghiệp vụ; không để domain import DynamoDB hoặc HTTP/SQS types.
- [ ] Chạy `go -C backend test -race -count=1 ./...`, `go -C backend vet ./...`, integration Local và `make build-lambdas`; xác nhận vẫn đủ 8 ZIP có tên hiện tại.
- [ ] Kiểm tra contract fixture và Terraform resource addresses không thay đổi; không đổi JSON/key/HTTP status trong PR này. Commit riêng các đợt chuyển module khi thực thi.

**Gate:** service độc lập AWS SDK, tests/integration pass, binary entrypoint giữ nguyên; deployment chưa bị ép đổi chỉ vì cấu trúc Go.

### P2A — Profile và RBAC phía backend

**Files:** tạo `backend/internal/identity/{principal,policy,profile}.go`, `identity/policy_test.go`, `adapters/dynamodb/profiles.go`, `transport/httpapi/auth.go`; thêm RBAC cases vào `backend/tools/local-http/`; sửa `infrastructure/terraform/environments/dev/main.tf` cho quyền đọc profile đúng resource.

**Interfaces:** `identity.Principal{Subject string, Role Role, Active bool}`; `identity.Authorize(actor Principal, action Action, ownerID string) error`; `ProfileStore.Get(ctx, sub) (Profile, error)`. Ma trận role trong thiết kế là hợp đồng; `ErrForbidden` → 403, truy cập link người khác của user thường tiếp tục 404 để giữ contract.

- [ ] Viết table tests `TestPermissionMatrix`, `TestDisabledAccountWithUnexpiredToken`, `TestMissingProfileDefaultsToUser`, `TestClientCannotAssignRole`; yêu cầu mọi action ngoài ma trận bị từ chối. Chạy để thấy fail trước implementation.
- [ ] Implement canonical role/profile và middleware/service policy; bootstrap profile cũ bằng conditional write, mặc định user. Không suy email thành owner và không phụ thuộc token claim cũ để thu hồi quyền ngay.
- [ ] Kiểm profile theo trạng thái hiện hành cho mọi API/job được bảo vệ; thêm test role bị hạ trong lúc JWT còn hạn và direct API invocation bỏ qua UI.
- [ ] Chạy unit/integration và contract tests với user/staff/analyst/admin; giữ Google PKCE và ownership tests cũ. Chỉ bật endpoint global khi policy đã được áp dụng.

**Gate:** không có endpoint mới chỉ “ẩn nút” mà bỏ kiểm quyền server; account cũ vẫn sở hữu đúng link cũ.

### P2B — Đăng ký và refresh qua Cognito

**Files:** tạo `docs/superpowers/specs/<ngày>-cognito-session-design.md`, `backend/internal/identity/session.go`, `adapters/cognito/session.go`, `adapters/dynamodb/sessions.go`, `backend/cmd/auth-session/main.go`; sửa `frontend/src/lib/auth.ts`, `components/AuthCallback.tsx`, `features/auth/*`, Cognito/HTTP API/frontend hosting Terraform, tests auth và CI build handler list.

**Interfaces đề xuất để chốt trong spec:** `GET /api/auth/login`, `GET /api/auth/callback`, `GET /api/auth/session`, `POST /api/auth/refresh`, `POST /api/auth/logout`. Callback xử lý code/PKCE/state/nonce; cookie opaque tham chiếu session server-side; session/refresh trả access token ngắn hạn cho API client. Những endpoint này là phương án mới, không giả vờ chúng đã có.

- [ ] Chốt bật Cognito native bên cạnh Google, email verification/recovery, username profile/reservation và luồng onboarding account cũ. Không thay pool vì username attribute khác SRS; không tự link account chỉ theo email.
- [ ] Viết failing tests callback state/nonce một lần, session expiry 7 ngày, single-flight refresh, race hai tab, Origin/CSRF, logout/revocation, disabled profile và response không cache. Session ID lưu dạng hash; token server-side mã hóa và scrub log.
- [ ] Implement backend callback/session và frontend caller trong chế độ opt-in; giữ callback cũ cho các release còn dùng SPA exchange trong thời gian chuyển đổi đã định.
- [ ] Cập nhật refresh 7 ngày và rotation bằng Terraform, đồng thời bỏ flow không tương thích `ALLOW_REFRESH_TOKEN_AUTH`; giữ access 15 phút, callback URLs và Google settings. Frontend không lưu refresh token dài hạn trong web storage.
- [ ] Kiểm CloudFront chuyển `/api/auth/*` đúng origin, forward cookie/Set-Cookie, CORS credentials đúng origin, response TTL 0; chạy auth interaction tests, fresh static build và Terraform mock tests.
- [ ] Smoke staging bằng người dùng thật có quyền: Google, email signup/verify, refresh, logout, đổi role; không lấy invocation Lambda với subject tổng hợp làm bằng chứng browser OAuth đã chạy.

**Gate:** SRS auth được ánh xạ thành Cognito rõ ràng, không có hệ thống password/JWT thứ hai; chỉ rollout sau khi chốt spec session và tương thích frontend release.

### P3A — Runtime tương thích và lifecycle

**Files:** sửa `backend/internal/shortener/{model,policy,service}.go`, `adapters/dynamodb/links.go`, `transport/httpapi/links.go`; tạo `backend/cmd/update-link/main.go`, `contracts/openapi.yaml`; sửa moderation ASL, HTTP API/CORS, build list và `frontend/src/features/links/*`.

**Interfaces:** `Link` thêm `ExpiresAt`, `Enabled`, `DeletedAt`, `AdminSuspendedAt`, `Revision`; moderation status vẫn độc lập. `SetEnabled(ctx, actor, code, enabled, expectedRevision)`, `UpdateExpiry(ctx, actor, code, expiresAt, expectedRevision)`, `SoftDelete(ctx, actor, code, expectedRevision)` trả Link hoặc typed error; admin suspension có action riêng, owner không được tự gỡ.

- [ ] Viết failing tests `TestDualReadExpiry`, `TestLegacyMissingFlagsRemainEnabled`, `TestClearExpiryDoesNotRestoreLegacyTTL`, `TestResolveAtExpiryBoundary`, `TestSoftDeletePreservesHistory`, `TestOwnerCannotUndoAdminSuspension`, `TestModerationCannotReactivateSuspendedLink`, `TestConcurrentUpdateRevision`; cả legacy/new route dùng cùng predicate.
- [ ] Implement vCompat trên **tất cả** reader/writer/aggregator/moderation trước khi bật field mới. Legacy thiếu enabled mặc định true; schema v2 ghi expiresAt=0 rõ ràng cho vĩnh viễn, không fallback TTL sau khi owner bỏ expiry. Resolve đọc nhất quán; 404 cho pending/rejected/disabled/deleted/expired theo contract hiện tại. Không thêm 301 hoặc cache stale.
- [ ] Thêm PATCH theo revision và soft delete; response/public DTO không lộ owner internals. Bổ sung `PATCH` vào CORS, routing và permissions/boundaries liên quan cùng PR.
- [ ] Kiểm alias và route collision trước `/api/*`, `/r/*`, probe paths và trang mới. Giữ `GET /{code}`; nếu thêm `/r/{code}`, route đó gọi thẳng cùng resolver để không tạo thêm redirect hop.
- [ ] Kiểm staging và ghi alias/version **vCompat** tối thiểu vào release matrix. Chưa cho backfill chạy khi còn runtime không hiểu soft delete/expiry mới.

**Gate:** khóa/xóa/hết hạn được mọi code path hiểu giống nhau; rollback target đã hiểu field mới.

### P3B — Filters, quota và idempotent create

**Files:** tạo `backend/internal/shortener/{filters,quota,idempotency}.go` và tests, `adapters/dynamodb/{quota,idempotency,catalog}.go`, `docs/refactor/access-patterns.md`; sửa frontend API/list workspace, HTTP CORS, CloudFront forwarding, DynamoDB module và Terraform tests.

**Interfaces:** `List(ctx, actor, LinkFilter, PageToken) (LinkPage, error)`; filter chuẩn hóa status/date range/destination host, `limit` 1–100. `Create(ctx, actor, CreateCommand{URL, Alias, Expiry, IdempotencyKey})` bind key với subject và payload; cùng key/cùng payload trả lại kết quả, payload khác → 409.

- [ ] Viết failing tests filter + cursor gắn owner/filter, page rỗng vẫn còn continuation, alias dài/case-sensitive, 61 lần tạo trong rolling 60 giây, quota race, `Retry-After`, retry create sau timeout và key không thuộc owner khác.
- [ ] Chốt access patterns trước index: owner/date query trên GSI hiện có; filter bổ sung có giới hạn evaluated items; global catalog dùng read index có sharding/date bucket, không Scan trong HTTP. Cursor giữ trạng thái merge/query và filter fingerprint; không giả tổng số trang chính xác.
- [ ] Implement sliding window 60/60s mặc định bằng conditional/CAS DynamoDB, prune theo clock thay vì chờ TTL; contention có retry hữu hạn và lỗi rõ ràng. Gate API throttle chung không thay quota theo user.
- [ ] Lưu idempotency result theo owner/key và payload hash; thời hạn đề xuất 24 giờ. Phối hợp quota reservation, link creation và trạng thái moderation để retry không tạo link/charge quota hai lần. Link pending khi StartExecution lỗi phải có cơ chế retry/reconciliation bền vững, không orphan vô hạn.
- [ ] Chỉ khôi phục header `Idempotency-Key` khi backend/CORS/CloudFront/frontend đã hỗ trợ cùng nhau; có regression cho lỗi CORS từng gặp. UI bỏ `listAllLinks` khỏi luồng tương tác và thực hiện pagination thật.
- [ ] Chạy service/integration, frontend list/create lifecycle tests và Terraform mock tests; dùng fixtures nhiều owner và filter ít match để chứng minh không tải hết catalog.

**Gate:** lifecycle có thể dùng độc lập, pagination/quota đúng trong concurrency, tạo link retry an toàn và moderation không bỏ sót.

### P3C — Backfill metadata có checkpoint

**Files:** tạo `backend/tools/migrate-link-metadata/{main,migration}.go`, `migration_test.go`, `docs/runbooks/link-metadata-migration.md`; sửa mapping expiry ở DynamoDB adapter sau khi gate vCompat đạt.

**Interfaces:** tool mặc định dry-run; input table/checkpoint, output count/hash theo owner/code và record lỗi; apply đòi mode tường minh. Mỗi conditional update chỉ chuyển LINK META `ttl → expiresAt` và schema version; không chạm TTL entity khác.

- [ ] Test fixture legacy TTL, vĩnh viễn, expired chưa bị TTL dọn, tombstone, concurrent PATCH và chạy lại migration hai lần; idempotent và không ghi đè revision mới.
- [ ] Snapshot/restore drill trong môi trường kiểm thử; ghi report kiểm kê alias route conflict và marker/raw khả dụng. Chỉ báo có backup khi restore thật đã được xác nhận.
- [ ] Dry-run, kiểm tra checksum/invariant; sau review mới apply theo batch có rate limit và checkpoint. Đối chiếu owner, code, target, expiry và totals trước/sau.
- [ ] Xác nhận rollback về vCompat hoạt động **sau khi bỏ ttl**; không dùng binary cũ chỉ đọc ttl. Giữ tombstone để code không được cấp lại.

**Gate:** migration được tiếp tục/chạy lại an toàn, không xóa metadata hoặc mất link vì đổi model; dữ liệu đã bị TTL xóa trước đó được ghi rõ ngoài khả năng tự khôi phục.

### P4A — Telemetry v2 và nhánh raw/enrichment

**Files:** tạo `contracts/events/{click-v1,click-v2}.schema.json`, fixtures, `backend/internal/analytics/{event,enrichment}.go`, `adapters/firehose/sink.go`, `backend/cmd/enrich-click/main.go`; sửa publisher/resolver/envelope decoder; thêm Terraform `modules/analytics_storage/*`, queue/rule/IAM và tests.

**Interfaces:** event v2 giữ ID ổn định qua retry; thêm version, code, owner, occurredAt, requestId, trusted client metadata; `Enrich(ctx, Click) (EnrichedClick, error)`, `RawSink.PutBatch(ctx, []EnrichedClick) []RecordResult`. v1 thiếu field được lưu unknown, không bịa lịch sử.

- [ ] Viết failing contract tests v1/v2, corrupt JSON, duplicate ID, spoofed forwarded IP, cùng click count thành công/raw thất bại, worker chết sau Firehose accept, per-record partial failure và enrichment timeout.
- [ ] Kiểm chứng viewer IP thực qua CloudFront/API Gateway bằng fixtures/staging; chốt GeoIP dataset/license/update version, UA parser và IP raw/masking. Không log token hoặc nguyên URL có dữ liệu nhạy cảm tùy ý.
- [ ] Fan-out bus hiện tại sang queue enrichment riêng; count consumer cũ tiếp tục là writer duy nhất. Chỉ ack record khi handoff durable thành công; retry failed records, cấu hình DLQ riêng và replay audit.
- [ ] Implement sink Firehose/S3: NDJSON có delimiter, buffering theo budget, ingestion-time partitions, error handling/alarms và lifecycle 90 ngày; test reads bằng event-time và dedupe eventId. Không coi Firehose API accept là bằng chứng raw đã hiện diện trong S3.
- [ ] Chạy shadow với fixture đã biết số click, reconcile accepted/published/count/raw unique; đánh dấu `analyticsStartedAt`, không gán tổng click cũ vào dữ liệu ngày mới.
- [ ] Chạy Go/contract/integration/Terraform tests và staging failure drill. Xác minh broker/publisher lỗi vẫn trả redirect đúng policy, có metric báo analytics thiếu; không tuyên bố zero loss.

**Gate:** có raw bền vững cho lịch sử mới, v1/v2 tương thích và nhánh raw có thể phục hồi độc lập nhánh count.

### P4B — Realtime và counter dưới tải

**Files:** sửa `backend/internal/realtime/*`, `adapters/dynamodb/{counter,connections}.go`, `transport/wsapi/*`, frontend realtime hook và link table; thêm `backend/tools/local-http/ws_e2e_test.go` cases và Terraform websocket/observability tests.

**Interfaces:** ticket owner subscription cấp qua endpoint riêng, không phát ticket cho mọi dòng khi list. Payload giữ code/clicks, thêm version/asOf nếu cần; một kết nối owner có thể cập nhật các link của người đó, không được subscribe owner khác.

- [ ] Test one-use ticket, active profile/ownership, cross-owner fan-out, reconnect snapshot, token/account expiry, bounded broadcast và notification fail không retry transaction đã commit.
- [ ] Chuyển UI sang một socket theo owner, reconcile theo trang đang xem; giữ per-link mode trong giai đoạn client release cũ còn hoạt động. Chốt deadline dừng mode cũ trong release matrix.
- [ ] Benchmark hot-link counter. Nếu cần sharding để đạt gate tải, dùng generation/cutover baseline + hash(eventId) shard; test duplicate/replay/concurrent cutover không làm tăng hai lần hoặc giảm tổng. Nâng compatibility floor của cả counter reader/writer trước cutover; tăng dedupe horizon trước khi cho replay dài hơn 15 ngày.
- [ ] Đo freshness riêng: live totals theo giây, dimensions/rollup theo batch; UI hiển thị `asOf`. Không gọi heatmap 10 phút là dữ liệu từng click tức thời.

**Gate:** số socket theo session/owner, quyền và totals giữ đúng; phương án counter đã được đo cho tải được duyệt.

### P5 — Jobs, rollup, backfill và anomaly

**Files:** tạo `backend/internal/operations/{job,lease,service}.go`, `analytics/{rollup,anomaly}.go`, `adapters/athena/*`, `adapters/dynamodb/{jobs,stats,alerts}.go`, `backend/cmd/analytics-job/main.go`, `statemachine/analytics-jobs.asl.json`, `infrastructure/terraform/modules/analytics_jobs/*`, SQL query templates versioned và fixtures.

**Interfaces:** `StartJob(ctx, actor, JobRequest) (JobID, error)`, `GetJob(ctx, actor, JobID) (Job, error)`; states queued/running/succeeded/failed/cancelled; lock key jobType/window/algorithmVersion. Stats snapshot có window, generation, source watermark, asOf và provisional/final.

- [ ] Test fixture chuẩn: 100 event, 10 event ID bị giao lại, IP lặp qua hai giờ, event đến muộn qua nửa đêm UTC, thiếu GeoIP. Assert exact unique/day, top 10 referrer, country/device totals và kết quả rerun không đổi.
- [ ] Test scheduled/manual cùng window, lease hết hạn rồi worker cũ tiếp tục, crash sau query trước commit, backfill ngày không còn raw, range không hợp lệ và role thiếu quyền. Worker cũ không ghi đè snapshot generation mới.
- [ ] Implement job registry/checkpoint và Step Functions ASL: Athena start/wait/result, update read model, lỗi và resume; không dùng goroutine/cron trong Lambda. Scheduler 10 phút, 00:05 UTC, 15 phút và weekly; retry/DLQ theo stage.
- [ ] Implement rollup bằng dedupe raw và overwrite snapshot, daily unique riêng hourly; time filter event-time + partition filter ingestion-time. Đề xuất cửa sổ late event 24 giờ, daily lúc 00:05 provisional; backfill cũ hơn chạy explicit trong raw retention.
- [ ] Compact giữ provenance, không đếm đồng thời file nguồn và file compacted; query chỉ đọc generation/prefix được manifest công bố. Lifecycle áp dụng cả raw/error/intermediate theo ngày nguồn để copy không kéo dài retention ngoài chủ đích.
- [ ] Chốt anomaly formula/sample minimum/volume trong spec P5; triển khai alert registry có baseline và giải thích. Alert chỉ đề xuất suspend; thao tác admin enforce ở service P3, không cho owner gỡ.
- [ ] Thêm global KPI/top-link snapshots và read models dimension có kích thước hữu hạn; UI requests chỉ Query/Get read model. Backfill không phát click event để tăng lifetime counter.
- [ ] Chạy fixture query trên test data, tests ASL/Terraform và staging replay/resume drill. Đối chiếu raw unique, rollup và live baseline, giải thích phần v1/legacy thiếu dimension.

**Gate:** jobs tái lập/resumable, không đếm trùng, không query lịch sử vượt retention và không cần scan raw lúc user mở dashboard.

### P6 — Analytics, admin, operations và export

**Files:** tạo `frontend/src/app/dashboard/{analytics,admin}/page.tsx`, `features/{analytics,admin,operations}/*`; thêm backend handlers analytics/admin/jobs/export, `contracts/openapi.yaml`, S3 export policies và các tests frontend/API/permission.

**Interfaces:** `GET /api/analytics/links/{code}`, `GET /api/admin/kpis`, `GET /api/admin/links`, `GET /api/admin/anomalies`, `GET /api/admin/health`; `POST /api/admin/jobs` và `GET /api/admin/jobs/{jobId}`; `POST /api/exports` → 202/jobId, tải file sau recheck quyền. Query params/window/response schema khóa trong OpenAPI trước code.

- [ ] Viết failing permission/API/interaction tests: regular user chỉ xem analytics của mình; staff đọc/export link toàn cục; analyst xem/export analytics; admin chạy job và suspend. Direct calls, jobId đoán được và role bị thu hồi đều phải bị chặn.
- [ ] Xây KPI active links, today clicks, top 10 links, 24h heatmap, referrer/country/device; hiển thị UTC window, thời điểm cập nhật, trạng thái provisional/thiếu dữ liệu legacy.
- [ ] Xây anomaly monitor và thao tác suspension riêng với audit actor/reason/time; owner toggle không thay trạng thái suspension. Operations UI có backlog/oldest-message age, service status/latency, manual trigger/backfill và job progress.
- [ ] Health không dùng một AWS dependency optional để đánh sập toàn bộ redirect. Public liveness trả tối thiểu; detailed readiness/queue/service diagnostics chỉ dành admin, có timeout và cache snapshot phù hợp.
- [ ] Export CSV/JSON async, filter cố định, CSV escaping/chống formula injection, file private và URL tải có hạn; không đưa raw IP vào export mặc định. Recheck role khi lấy URL; file/link cũ có thời hạn hữu hạn và audit.
- [ ] Chạy frontend interaction tests, lint/typecheck/fresh static build, backend permission/integration và Terraform tests. Smoke staging bao gồm browser auth thật, role-based navigation, export và job/backfill.

**Gate:** toàn bộ FR-06 có luồng UI/backend kiểm chứng, admin thao tác được qua ứng dụng mà không cấp AWS console credentials cho trình duyệt.

### P7 — NFR và phát hành có rollback

**Files:** sửa `tests/load/redirect.k6.js` và tests, `tests/e2e/*`, `.github/workflows/{ci,deploy}.yml`, `scripts/publish-frontend.sh`, Terraform observability/hosting; tạo runbooks release, rollback, queue replay, restore và retention.

**Interfaces:** release manifest pin backend versions, frontend release prefix, config/schema/algorithm versions, vCompat minimum và feature flags; báo cáo test phân biệt functional, load, resilience và security.

- [ ] Nâng load test hiện tại từ vài VU có sleep sang arrival-rate workload phù hợp bài đo RPS; vẫn không follow redirect. Test root code và `/r/`, hot/random code, active/disabled/expired, warm/cold và peak/sustained theo budget.
- [ ] Chạy 100 → 500 → 1.000 → 5.000 RPS ở staging đã chuẩn bị; ghi p50/p95/p99, error/429, Lambda duration/concurrency, publish latency, DDB throttles, queue lag, raw freshness và cost/triệu click. Bài đo mock/local không thay bài public.
- [ ] Fault drill: publish/enrichment/raw delivery lỗi, DDB throttling, expired ticket/token, queue replay, duplicate delivery và scheduler overlap. Báo mất event trước broker như một giới hạn có số đo; không giấu bằng số click cache.
- [ ] Kiểm headers trên HTML/API/error/redirect; CSP không phá static UI; HTTPS viewer/origin, TLS version thực tế, spoofed headers, auth/CSRF và data access. Ghi ngoại lệ TLS 1.3 nếu dịch vụ/chặng hiện tại không đáp ứng.
- [ ] Bật immutable frontend release prefix/manifest và rollback pointer đã kiểm thử; giữ callback/API backward compatibility trong khoảng hỗ trợ client cũ.
- [ ] Chạy `make check`, integration Local và staging E2E; lưu output/report theo release. Terraform saved plan phải không replace pool/table/state ngoài chủ đích; bootstrap boundary preflight trước apply.
- [ ] Sau khi có yêu cầu phát hành riêng: triển khai canary, theo dõi alarms và capability flags; drill rollback backend về vCompat + frontend manifest + job generation, không phục hồi raw/metadata bằng thao tác xóa dữ liệu mới.

**Gate:** từng FR trong matrix trỏ tới test và bằng chứng release. Các số NFR gốc đạt bằng đo thực, hoặc có quyết định sửa SRS/SLO AWS được review rõ; không ghi “hoàn thành toàn bộ SRS” khi gate này chưa giải quyết.

## 4. Việc có thể bắt đầu sau khi duyệt kế hoạch

Đợt đầu nên chỉ lấy **P0 và P1**: cố định baseline hiện tại, chốt SRS áp dụng, thêm contract fixtures cần thiết và tách module không đổi hành vi. Sau khi review đợt này mới triển khai P2/P3; đây là nơi schema và quyền bắt đầu thay đổi. P4–P6 cần raw analytics mới nên không thể lấy dữ liệu tổng click cũ để làm tắt.

Trong lần lập kế hoạch này đã thêm hai tài liệu Markdown và rà soát độc lập thiết kế. Chưa scaffold module, thêm dependency, sửa code, chuyển dữ liệu, commit hoặc deploy.
