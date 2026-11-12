# Kế hoạch xây Shrnk Go Gin ở local rồi triển khai AWS

> **Đã được thay thế theo quyết định ngân sách ngày 12/11/2026.** Dùng [kế hoạch Go/Gin Lambda dưới $20](2026-11-12-go-gin-budget-plan.md) và [thiết kế tương ứng](../specs/2026-11-12-go-gin-budget-architecture.md). Không thực thi các bước dựng RDS/Atlas/ElastiCache/Amazon MQ trong bản cũ này.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox syntax. Đây là kế hoạch chuẩn bị; các bước deploy chỉ thực hiện sau local gate và yêu cầu phát hành riêng.

**Goal:** Xây hệ thống theo SRS Go/Gin mới, giữ frontend Next.js và tài nguyên AWS hiện tại, sau đó đưa cùng ứng dụng container đã kiểm chứng lên kiến trúc AWS mới.

**Architecture:** Backend mới là REST API Gin với MySQL, MongoDB, Redis và RabbitMQ. API, worker enrich, worker aggregate và beat là các process/container độc lập; shared service layer không phụ thuộc Gin. AWS đích là ECS Fargate, RDS MySQL, Atlas trên AWS, ElastiCache và Amazon MQ.

**Tech Stack:** Go/Gin, Next.js hiện tại, SQL thuần, MongoDB official driver, Redis-compatible client, AMQP, Docker Compose, Terraform. Pin versions theo compatibility matrix trước implement; không dùng `latest` làm release version.

**Spec:** [SRS mới](../../requirements/SRS_URL_Shortener_Go_Gin.docx) và [thiết kế ứng dụng/AWS](../specs/2026-11-12-go-gin-aws-architecture.md). Hai tài liệu này thay định hướng serverless/Lambda của kế hoạch trước.

## Global Constraints

- Hoàn thiện và kiểm chứng local trước; giữ nguyên AWS v1. Không apply/destroy Terraform, đổi DNS, mutate DB/queue cloud hoặc publish frontend trong các pha local.
- Giữ Next.js. Go chỉ phục vụ REST/redirect, không có `static/`, `templates/`, `migrations/`. Schema SQL ở `db/`, config ở `config/config.yaml`.
- `GET /r/{code}` trả **301**; cache sliding TTL mặc định **1 giờ**, clamp theo expiry; tracking và logging phải bất đồng bộ, bounded và quan sát được loss.
- `enrich.fast` và `aggregate.bulk` là queues riêng; workers có scale/concurrency/retry riêng; raw upsert và rollup phải idempotent.
- Structured logs bắt buộc có `request_id`, `task_id`, `latency`, `retries`.
- Không lấy p95/RPS/retention của SRS cũ làm requirement mới. Chốt SLO, retention, quota, late-event và token lifetimes bằng ADR/config trước gate liên quan.
- Giữ thay đổi đang có trong working tree. Xây `backend-v2/` trong worktree thực thi riêng; không xóa backend/frontend/infra cũ để scaffold cho nhanh.
- SQL không auto-migrate khi server/worker khởi động; CLI/schema runner là thao tác riêng có version, checksum và explicit mode.

## Review Focus

1. Toggle/expiry/delete/admin suspension đua cache hit/fill/warm: không trả URL bị vô hiệu sau mutation thành công; test P2/P3.
2. Claim role, refresh replay, staff đồng thời là owner, jobId và cache ops trái quyền: backend chặn đúng bất kể UI; test P1/P5.
3. Mongo đã upsert nhưng ack/Redis update lỗi, message giao trùng hoặc bot classifier đổi: không đếm hai lần, raw không mất; test P4/P5.
4. Beat rolling restart, backfill trùng window, daily unique và event đến muộn: lease/generation bảo vệ snapshot; test P5.
5. Broker/Redis/log sink lỗi, channel đầy hoặc process bị dừng: redirect/degraded policy rõ, retry hữu hạn, drop/freshness đo được; test P3/P4/P6.

## P0 — Chốt hợp đồng và baseline

**Files:** `backend-v2/docs/openapi.yaml`, `docs/architecture-decisions.md`, `docs/event.schema.json`, `docs/version-matrix.md`, `docs/local-acceptance.md` trong backend mới khi bắt đầu implement. Tài liệu hiện tại mới mô tả chúng, chưa scaffold.

**Interfaces:** định nghĩa API DTO/typed errors, `ClickEvent{EventID, Code, LinkID, OwnerID, OccurredAt, IP, UserAgent, Referer, RequestID, SchemaVersion}`; job request với loại/window/algorithmVersion; UTC là storage window, UI ghi rõ timezone.

- [x] Đọc SRS mới và khảo sát sự khác biệt với backend Lambda/Next.js hiện tại.
- [x] Thiết kế AWS và ranh giới local/staging/cutover bằng tài liệu, chưa provision.
- [ ] Cố định baseline gồm HEAD và diff chưa commit, tạo worktree lúc thực thi và bảo toàn các audit/plan cũ.
- [ ] Pin Go tương thích Gin, MySQL/Mongo/Redis, RabbitMQ cùng major được Amazon MQ hỗ trợ; ghi version/digest và supported command set. Go hiện tại 1.26.10 không cần hạ xuống vì SRS cũ từng ghi 1.22+.
- [ ] Chốt đề xuất auth/role, 301/no-store, bot flag, tracking loss policy, retention/quota/SLO và legacy alias 32 ký tự. Chưa có retention thì không tạo TTL index xóa raw.
- [ ] Viết OpenAPI cho auth, CRUD/filter, redirect, analytics, health, jobs/export/cache ops; lấy request/response fixtures hiện có làm bằng chứng tương thích, không sao chép Lambda envelope.

**Gate:** người triển khai phân biệt được requirement SRS, đề xuất mới và dữ liệu legacy; không còn phải đoán public contract trước khi viết code.

## P1 — Nền local, schema và auth

**Files:** tạo `backend-v2/{go.mod,config/config.example.yaml,db/*.sql,docker/*}`, `cmd/{server,cronjob,cli}/main.go`; `internal/{initialize,models,repo,service,middlewares,routers,controller}/*`, `pkg/{setting,logger,response}/*`; test cùng package và `tests/integration/`.

**Interfaces:** `setting.Load(path string) (Config, error)`; `initialize.Open(ctx, Config) (*Dependencies, error)` và Close; auth `Register`, `Login`, `Refresh`, `Logout` nhận context/DTO, trả token/profile DTO. Controller chỉ map Gin request/response.

- [ ] Viết failing tests config sai/thiếu, schema init lại, unique email/username/code phân biệt hoa thường, role tự nâng, disabled account, password verify và refresh replay; chạy để xác nhận fail.
- [ ] Compose MySQL/Mongo replica set/Redis/Rabbit với health checks, volumes local và TLS/config profile tương ứng; chặn default config trỏ tới host AWS thật. Pin Rabbit queue type từ đầu.
- [ ] SQL thuần tạo users/roles/links/domains/refresh_sessions/job_runs/outbox; schema CLI apply explicit theo checksum và lock, không ORM auto-migrate. `links` có revision, expiry, deleted_at, admin_suspended_at và code đủ chứa alias cũ.
- [ ] Implement pools, dependency injection, error envelope, log context và graceful lifecycle. `global/` không là nơi giữ request/session mutable state.
- [ ] Implement JWT/native auth theo ADR P0, refresh cookie/CSRF, role policy và schema seed mặc định user. Giữ Cognito v1 nguyên trạng; không gọi nó để local login hoạt động.
- [ ] Chạy `go test -race ./...` và integration trong `backend-v2`; kiểm schema/indices bằng DB thật trong Compose. Các lệnh này chỉ tồn tại khi P1 đã tạo module.

**Gate:** local khởi động tái lập; schema có kiểm soát; native auth/RBAC và refresh dùng được mà không có AWS credentials.

## P2 — Link CRUD, filters và quota

**Files:** `internal/service/{links,permissions}.go`, `internal/repo/{mysql_links,redis_quota}.go`, `controller/links.go`, `routers/routes.go`, SQL indexes và tests; frontend `src/lib/api.ts`, `features/links/*` trong worktree thực thi.

**Interfaces:** `CreateLink(ctx, Principal, CreateLinkInput) (Link, error)`; `ListLinks(ctx, Principal, LinkFilter, PageRequest) (Page[Link], error)`; `UpdateLink`, `SoftDeleteLink`, `SuspendLink` có expectedRevision. Rate limiter theo User/IP trả allowed/retryAfter; 429 chỉ khi quota vượt, dependency outage có lỗi riêng.

- [ ] Failing tests owner/cross-owner, staff owner vẫn read-only, expiry boundary, alias collision, revision race, pagination/filter status/expiry và quota concurrency/spoofed client IP.
- [ ] Implement parameterized MySQL queries và SQL indexes theo access patterns; page/limit được bound, filter ngày/hạn dùng rõ. Giữ soft-delete tombstone/code; URL validator tái sử dụng quy tắc an toàn đã có nhưng tách AWS types.
- [ ] Implement Redis Lua limiter atomically với TTL theo cấu hình User/IP; trust proxy chain local/CloudFront có contract riêng. Redis outage không tự cho create không giới hạn.
- [ ] Implement cache-bypass fence/revision và outbox repair cho mutation trước khi bật cache ở P3; owner không tự gỡ admin suspension; admin warm/invalidate cũng đi qua service policy.
- [ ] Rerun unit/integration, concurrent API cases và frontend create/list/update/delete. Không dùng `listAllLinks` cho bảng tương tác.

**Gate:** catalog/lifecycle và permission đúng trên DB thật; một cache DEL đơn lẻ không được coi là đủ để nghiệm thu invalidation.

## P3 — Resolver và nền tracking/logging không chặn

**Files:** `service/{resolver,tracking}.go`, `repo/{redis_links,rabbit_publisher}.go`, `controller/redirect.go`, `pkg/logger/async.go`, resolver/publisher tests và `tests/load/redirect.k6.js` cho bản mới.

**Interfaces:** `Resolve(ctx, code string) (ResolvedLink, error)`; `TrackingSink.TryEnqueue(ClickEvent) bool`; `Publisher.Publish(ctx, ClickEvent) error`; publisher owns buffer/confirm/reconnect/drain. Gin adapter giữ nguyên 301 contract, không truyền Gin context vào background goroutine.

- [ ] Failing tests cache hit/miss/refresh TTL1h, clamp expiry, disabled/deleted/suspended, hit lúc cache-fill/warm và mutation chạy đua, query MySQL timeout, Redis outage.
- [ ] Implement 301/no-store, cache payload có state/expiry/revision và fenced refill; invalid/missing/inactive không cache thành redirect. Warm/invalidate không chạm quota/session keys.
- [ ] Implement bounded tracking/log queue, stable event ID, publisher confirms, retry deadline và shutdown drain. Test broker down/full buffer/process shutdown: redirect theo policy P0, drop metric rõ, không goroutine leak/unbounded memory.
- [ ] Chạy load local không follow redirect, tách warm/cold/Redis-failure cases; lưu origin latency/p95/p99, broker wait và tracking drop. Không dùng router microbenchmark để khẳng định end-to-end latency.

**Gate:** resolver đạt SLO local đã chốt; mọi failure mode được đo, không tuyên bố zero loss cho buffer RAM.

## P4 — Enrichment, raw upsert và now counters

**Files:** `cmd/cronjob/*`, `service/{enrichment,classification}.go`, `repo/{mongo_clicks,redis_now,rabbit_consumer}.go`, index CLI, fixtures UA/GeoIP/bot, retry/DLQ tests.

**Interfaces:** `Enrich(ctx, ClickEvent) (EnrichedClick, error)`; `ClickRepo.Upsert(ctx, EnrichedClick) (Inserted bool, err error)`; consumer xử lý ack/nack độc lập record. Raw upsert dùng event_id unique và `$setOnInsert`; thay classification có version qua job riêng.

- [ ] Failing tests duplicate/reordered messages, poison payload, partial batch write, GeoIP unknown, bot flag, crash sau Mongo write trước ack/Redis update và broker reconnect.
- [ ] Implement enrich.fast consumer, prefetch/concurrency bound, retry queues/DLQ, confirms cho handoff. Giữ raw bot flag nhưng loại bot khỏi analytics mặc định theo ADR; lưu version GeoIP/parser.
- [ ] Implement Redis now bucket/dedupe + reconciliation; Mongo raw là source of truth, không giả có transaction chung Mongo–Redis. Không skip repair chỉ vì raw đã tồn tại.
- [ ] Kiểm fast queue vẫn xử lý khi aggregate worker đang giữ job lớn; test restart consumer, retry DLQ có audit, drift/rebuild now và memory/goroutine ổn định.

**Gate:** raw clicks mới có provenance/dedupe, bot policy và realtime read model có thể phục hồi.

## P5 — Jobs, aggregation, backfill và admin APIs

**Files:** `service/{jobs,rollup,anomaly,exports,cache_ops,health}.go`, `repo/{mysql_jobs,mongo_stats,mongo_queries}.go`, `cmd/cronjob/beat.go`, SQL job/outbox và Mongo indexes; API controllers cho admin/analytics/jobs/export.

**Interfaces:** `EnqueueJob(ctx, Principal, JobRequest) (JobID, error)`; `RunWindow(ctx, JobLease) (Snapshot, error)`; `GetJob(ctx, Principal, JobID) (Job, error)`; snapshot có grain/window/generation/algorithmVersion/asOf. Lease có fencing token; job registry/outbox bảo vệ enqueue sau SQL commit.

- [ ] Failing fixture tests 100 unique event + duplicates, IP qua nhiều giờ, bot included/excluded, late arrival qua nửa đêm UTC; assert exact totals, daily unique và top10.
- [ ] Failing tests hai beat, scheduled/manual cùng window, expired lease worker vẫn chạy, crash sau tính trước commit, backfill ngoài retention, export/job ID của người khác và cache ops trái quyền.
- [ ] Implement beat enqueue job theo YAML, workers aggregate.bulk, checkpoints/cancel/resume; thay snapshot idempotent bằng generation, daily unique tính riêng, không tăng lifetime baseline qua backfill.
- [ ] Implement hourly/daily KPIs, heatmap, anomaly theo công thức/tham số P0. Compact không để query đếm cả source và bản compacted; retention áp dụng cả raw/export/copies theo policy.
- [ ] Implement healthz/readyz/detailed admin health, job pages API và export adapter local/S3. CSV escaping/formula protection; permissions check lại trước lấy URL/file; cache ops bounded và audited.
- [ ] Chạy fixtures Mongo aggregation, job restart/replay tests và permission matrix đầy đủ trước kết nối UI.

**Gate:** thống kê tái lập, job vận hành có audit/progress, backfill và export không vượt quyền hoặc retention.

## P6 — Next.js và nghiệm thu local toàn hệ

**Files:** sửa frontend auth/API/config/hooks; thêm `features/{analytics,admin,operations}/*` và pages cần thiết, frontend interaction tests, local browser e2e và `backend-v2/docs/local-acceptance.md`.

**Interfaces:** frontend REST client theo OpenAPI mới, refresh single-flight, role-based navigation; now polling 3–5 giây đề xuất với asOf. Frontend v1 build/config vẫn có thể tái tạo riêng để vận hành cloud cũ.

- [ ] Failing UI/API tests register/login/refresh/logout, owner lifecycle, readonly staff/analyst, admin suspension/cache ops/jobs/backfill/export, expired session và degraded metrics.
- [ ] Giữ Next.js, dùng style/components hiện tại khi phù hợp; bỏ phụ thuộc bắt buộc Cognito/WebSocket config trong build v2, không phá build v1. API base URL là cấu hình; browser local dùng local API.
- [ ] Chạy Go race/vet/integration, frontend lint/typecheck/fresh build/tests, browser e2e với Compose. Test real local auth, không thay bằng context subject giả lập.
- [ ] Fault/drain/load/restore drill bằng dữ liệu test: tắt broker/Redis/Mongo, restart beat/worker, SIGTERM, queue backlog; chốt capacity/buffer/pool/retention và chụp acceptance evidence.
- [ ] Tạo candidate manifest pin image digest, config/schema/index/event versions và frontend release. Chưa push ECR, apply AWS hoặc publish website chỉ vì local pass.

**Gate:** luồng SRS end-to-end local đạt và có bằng chứng; chỉ lúc này chuyển sang chuẩn bị AWS staging.

## P7 — AWS staging và cutover sau local gate

**Files dự kiến:** roots Terraform mới `infrastructure/terraform/environments/{v2-staging,v2-production}/`, modules `v2_*`, workflow ECR/ECS riêng, Atlas provider config, runbooks estimate/restore/migration/cutover/rollback. Không sửa state root cũ thành root mới.

- [ ] Trước provisioning: estimate theo region và footprint trong thiết kế; kiểm supported engines, Redis/Valkey parity, Atlas dedicated/PrivateLink subscription, MQ quorum/version và metrics collector. Không mặc định all-AWS first-party hoặc miễn phí.
- [ ] Sau yêu cầu triển khai riêng, tạo namespace/state/roles mới và staging hostname. ECR images dùng digest; schema/index là one-off task có quyền riêng; boot API không tự DDL.
- [ ] Test managed dependencies với cùng suite local; test CloudFront cookie/auth/no-cache, ALB origin guard, Atlas ports/DNS/TLS, broker confirms/failover và metrics queue qua management API tối đa mỗi phút.
- [ ] Roll ECS có circuit breaker/health gate, worker drain và beat lease; frontend release prefix/manifest bất biến. Test app rollback trong schema compatibility floor.
- [ ] Rehearse DynamoDB→MySQL link/owner/expiry import và legacy click baseline có asOf; đối soát tổng cuối sau v1 queue/traffic drain. Tài khoản Cognito cần identity mapping/onboarding, không tự phát password. Chốt write freeze hoặc delta sync trước chuyển traffic.
- [ ] Chỉ cutover khi checksums, ownership, legacy root links, cache state, stats freshness và browser auth đạt. Canary read-only trước mở writes nếu chưa thử reverse sync/identity failback; sau mở writes ưu tiên rollback release trong v2. Không hứa native account mới có thể đăng nhập vào v1 Google-only. Giữ v1 đến khi có yêu cầu ngừng riêng.

**Gate:** hạ tầng mới phù hợp ứng dụng đã hoàn thiện và được kiểm chứng tại AWS; local success không tự biến thành quyền deploy hoặc xóa tài nguyên cũ.

## Trạng thái bàn giao

Lần này hoàn thành việc đọc SRS mới, thiết kế ứng dụng/AWS và kế hoạch local trước. Chưa scaffold backend-v2, cài dependency, sửa code sản phẩm, chạy migration hoặc thay đổi AWS. Bộ kế hoạch serverless trước được thay bằng bộ tài liệu mới này.
