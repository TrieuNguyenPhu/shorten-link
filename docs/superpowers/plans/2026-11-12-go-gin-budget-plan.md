# Kế hoạch Go Gin local trước với AWS dưới 20 USD mỗi tháng

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans task-by-task. Đây là kế hoạch chuẩn bị, không tự cho phép deployment. Giữ các quyết định ngân sách và local-first trong toàn bộ quá trình.

**Goal:** Đáp ứng nghiệp vụ SRS mới bằng Go/Gin + Next.js, dùng adapter serverless ngay từ local và đưa lên AWS sau local gate trong workload envelope dưới $20/tháng.

**Architecture:** API Gin chạy local HTTP và Lambda HTTP API v2 qua entrypoint riêng; use cases dùng chung. DynamoDB giữ core/raw/analytics, S3 giữ batch archive/export, SQS tách fast/bulk, Scheduler thay beat cloud. Chưa có Redis/MySQL/Mongo/Rabbit hoặc container server thường trực trên AWS.

**Tech Stack:** Go/Gin, Next.js static, DynamoDB Local, ElasticMQ, filesystem object adapter, Lambda ARM64, HTTP API, DynamoDB/S3/SQS/Scheduler, Terraform.

**Spec:** [Kiến trúc và chi phí hiện hành](../specs/2026-11-12-go-gin-budget-architecture.md); [SRS gốc](../../requirements/SRS_URL_Shortener_Go_Gin.docx) áp dụng cùng các điều chỉnh hạ tầng đã được người dùng cho phép.

## Global Constraints

- Budget mục tiêu dưới $20/tháng cho v2, chưa tính domain/v1; không phải hard billing cap và không dùng Free Tier/credit làm điều kiện để phương án khả thi.
- Local functional/fault/contract tests xong trước deployment; giữ nguyên AWS v1 và working-tree changes. Dùng worktree lúc bắt đầu thực thi; chưa scaffold sản phẩm trong bước lập kế hoạch.
- Giữ Next.js, native auth/JWT/refresh/RBAC, owner lifecycle, URL validation, alias/case/expiry, soft delete, analytics/jobs/export/health. Giữ `/r/{code}` 301/no-store, legacy route qua adapter tương thích.
- Giữ config YAML và layout Gin của SRS; không Go templates/static/migrations. Bỏ SQL `db/` không còn engine, version data schema bằng access patterns và bootstrap CLI explicit.
- Không cache URL một giờ mặc định; đọc DynamoDB nhất quán để không stale lifecycle. Warm URL cache được thay bằng ops analytics read model, ghi rõ phụ lục SRS.
- Không goroutine/channel publisher hoặc cron thường trực trên Lambda. Enqueue SQS có deadline trước response, heavy processing async; publish lỗi không được che giấu và không hứa zero loss.
- Fast/bulk queues + DLQ riêng; raw/counter transaction và job generation phải idempotent. Không Scan toàn bảng trong HTTP; không dùng Athena trên mỗi lần tải UI.
- Đề xuất raw 30 ngày/log 7 ngày/export 24h; pin policy trước bật TTL. Core metadata link không bị TTL xóa.

## Review Focus

1. Expiry/delete/suspension và legacy defaults phải đúng khi đọc trực tiếp, owner không gỡ admin lock — P2.
2. Quota/JWT/role/profile/refresh replay và export/job đoán ID không vượt quyền — P1/P2/P4.
3. Event giao lại, raw transaction commit rồi Lambda timeout, late GSI data và backfill không tăng double — P3/P4.
4. Local runner và Lambda adapter khác vòng đời, queue partial failure và handler return không làm mất I/O cần thiết — P0/P3/P5.
5. Polling, item/index size, DDB transaction units, retries, logs và backfill không vượt cost envelope âm thầm — P4/P5/P6.

## P0 — Baseline, phụ lục SRS và contract

**Files:** dự kiến `backend-v2/docs/{srs-adaptation,openapi,access-patterns,event-schema,acceptance}` và config schema; bảo toàn backend/infra cũ.

- [x] Chốt quyền đổi hạ tầng, budget dưới $20 và local-first từ người dùng; lấy giá Singapore và dựng calculator offline.
- [ ] Cố định HEAD + dirty diff, tạo worktree thực thi. Giữ docs/audit có sẵn, không reset/stash hoặc commit chung ngoài phạm vi.
- [ ] Pin Go/Gin, AWS SDK, DynamoDB Local/ElasticMQ; chốt OpenAPI, roles, queue/event IDs, cursor filter binding và config endpoint guards. Giữ cấu trúc SRS trong backend-v2, không tạo package rỗng chỉ cho đủ cây.
- [ ] Chốt adaptation matrix: SQL/Mongo→DynamoDB; Redis→direct reads/quota/now read model; Rabbit→SQS; cloud cron→Scheduler; cache ops→read model ops. Không xây adapter bốn DB cũ như dependency bắt buộc.

**Gate:** không còn nhầm requirement nghiệp vụ với stack đã được thay; local và cloud sẽ dùng cùng biên service/repository/queue.

## P1 — Nền local, auth và adapter Lambda

**Files:** `backend-v2/cmd/{server,lambda-api,cronjob,cli}`, `internal/{controller,routers,middlewares,service,models,repo,initialize}`, `pkg/{setting,logger,response}`, `config/`, `docker/`, auth/config/router tests.

**Interfaces:** `setting.Load(path) (Config,error)`; `NewRouter(Dependencies) *gin.Engine`; services nhận `context.Context` và DTO, không Gin/APIGateway/SQS event types. HTTP API v2 adapter gọi cùng router; credentials/endpoint AWS được inject qua config.

- [ ] Failing tests config không trỏ cloud từ local, request validation/error envelope, role escalation, password/refresh/revocation/disabled profile, Origin/CSRF và local-vs-HTTP-v2 parity.
- [ ] Compose DynamoDB Local + ElasticMQ, local worker/beat runner, filesystem object adapter và dummy credentials. Bootstrap tables/indexes qua CLI explicit, không tự mutate schema khi server startup.
- [ ] Implement native auth trên Core với unique email/username reservations, refresh/session records; signing key YAML local/SSM reference cloud, không tự link Cognito user theo email.
- [ ] Run Go race/vet, local integration và API contract fixtures; frontend v1 vẫn build bằng config cũ, v2 không bắt buộc Cognito/WebSocket endpoints.

**Gate:** backend local và Lambda entrypoint compile/test cùng service, chưa có AWS request từ local tests.

## P2 — Lifecycle, pagination, quota và resolver

**Files:** `service/{links,resolver,permissions,quota}`, `repo/dynamodb_{core,catalog,quota}`, HTTP controllers, frontend link workspace và tests.

**Interfaces:** `Create`, `List`, `Patch`, `SoftDelete`, `Suspend` nhận actor + revision; `Resolve(ctx,code)` trả target hoặc typed error; `CheckQuota(ctx,subject,ip,now)` trả allowed/retryAfter.

- [ ] Failing tests legacy code/alias 32 ký tự/case, permanent/expiry boundary, role/ownership, admin suspension, optimistic revision, concurrent quota và cursor gắn owner/filter; sparse page có continuation vẫn hoạt động.
- [ ] Implement conditional writes, idempotency create, quota rolling window hữu hạn, profile policy và strong link reads. Không dùng TTL để coi link đã hết hạn hoặc quota đã reset.
- [ ] Giữ 301/no-store cho `/r`, tương thích root route cũ; không giữ URL lookup cache stale. UI dùng cursor thực, không listAllLinks; cache-specific ops chỉ hiển thị capability read-model mới.
- [ ] Run integration trên DynamoDB Local, frontend lifecycle và request/response parity qua Lambda adapter.

**Gate:** core nghiệp vụ đúng không cần Redis/SQL; mọi route cloud-forwarding/CORS sau này có contract rõ.

## P3 — SQS tracking, raw dedupe và now

**Files:** `cmd/lambda-enrich`, `service/{tracking,enrichment}`, `repo/{sqs,dynamodb_clicks,dynamodb_counters}`, event schema/fixtures, partial-batch tests.

**Interfaces:** `EnqueueClick(ctx,ClickEvent) error`; `Enrich(ctx,ClickEvent) (EnrichedClick,error)`; `RecordClick(ctx,EnrichedClick) (Applied|Duplicate,error)` là transaction raw + counters. Lambda mapping xử lý AWS envelope ngoài service.

- [ ] Failing tests stable ID, duplicate/partial batch, crash sau transaction trước ack, bot policy, invalid/over-size headers, source IP trust chain, expiry/delete sau lúc event được chấp nhận, SQS timeout và return không chờ background I/O.
- [ ] Implement bounded deadline SQS publish trước response; ưu tiên redirect khi publish lỗi kèm metric. Local runner có thể poll queue; cloud Lambda chỉ nhận batch qua event source mapping.
- [ ] Implement raw conditional put/counter transaction, GSI time bucket, TTL policy và enrich versions. Không tăng counter chỉ vì retry hoặc backfill; raw event ≤2 KiB trong budget envelope, normalization có flag.
- [ ] Set queue visibility/retry/DLQ/partial failures/concurrency trong spec Terraform, giữ fast riêng bulk. Worker GeoIP artifact/version không làm API package nặng.

**Gate:** events sau broker bền vững và idempotent; không tuyên bố zero loss trước SQS hoặc “hoàn toàn không chặn network”.

## P4 — Jobs, rollup, admin/export và Next.js

**Files:** `cmd/lambda-jobs`, `service/{jobs,rollup,anomaly,export,health}`, `repo/{dynamodb_analytics,s3}`, local object adapter, frontend `features/{analytics,admin,operations}` và tests.

- [ ] Failing fixtures duplicate + bot + cùng IP qua nhiều giờ + late arrivals; assert daily unique, top10/heatmap/KPI và rerun không đổi. Test lease/generation, overlapping/manual/backfill, expiry of raw/dedupe và cross-owner job/export.
- [ ] Implement time-bounded Query GSI theo window/shard với correction watermark; minute/hour/day snapshots và live day counters. Giới hạn dimension/item size và evaluated pages; ghi metrics resource use để đối chiếu WRU/RRU model.
- [ ] Scheduler contract gửi deterministic window job vào bulk SQS; manual job/outbox atomic, checkpoint/resume từng page/day dưới deadline Lambda. Backfill không phát lại click để tăng lifetime.
- [ ] Export/archive theo batch vào S3, TTL theo policy; không object/click. Raw archive có event ID, không query both raw và archive để đếm trùng; UI chỉ đọc read model có asOf.
- [ ] Implement Next auth/role navigation, analytics/jobs/health/read-model ops, polling theo active tab/session, pause hidden tab; CSV/JSON export có scope/escaping và kiểm quyền trước tải.
- [ ] Run full local browser e2e, Go/queue integration, frontend lint/typecheck/build/tests và fault/restart/drain/restore scenarios.

**Gate:** các nghiệp vụ SRS mới có luồng end-to-end; emulator limitations và SRS adaptations ghi rõ, chưa deploy AWS.

## P5 — Benchmark local và kiểm chứng budget

**Files:** load/acceptance fixtures, `docs/costs/*`, candidate manifest và deployment acceptance report.

- [ ] Đo warm/cold API, bcrypt/auth mix, worker/job duration, payload/index sizes, transaction units, queue requests/retries, log volume, polling/egress và backfill resources; không lấy số giả định trong calculator làm benchmark.
- [ ] Chạy calculator với kết quả đo và workload mục tiêu. Gate deploy target ≤$15 trước dự phòng; nếu mô hình vượt, giảm footprint/frequency/polling hoặc chốt lại workload/budget trước khi thêm dịch vụ.
- [ ] Pin image/binary/config/schema/event versions, loại secret khỏi artifacts; source/fixtures local không sử dụng production v1.

**Gate:** có capacity/cost envelope tái lập; chưa có cam kết hard cap hoặc bằng chứng AWS HA/latency từ local tests.

## P6 — AWS sau local gate, giữ v1 và migration

**Files dự kiến:** Terraform root/state/roles namespace budget-v2 riêng, HTTP API/Lambda/DynamoDB/SQS/S3/Scheduler/hosting/observability modules, CI OIDC và runbooks.

- [ ] Sau yêu cầu triển khai riêng: reviewed plan chỉ có service budget-v2, không VPC/NAT/ALB/RDS/MQ/Atlas/ElastiCache/Provisioned Concurrency hoặc paid WAF mặc định. Không thay state/module addresses v1 trong PR này.
- [ ] Deploy staging giới hạn, test IAM/TTL/retry/DLQ/S3 signed URLs, viewer IP, Cookie/Set-Cookie/CORS/no-cache và browser auth thực; kiểm số đo với calculator trước production.
- [ ] Tạo log retention 7 ngày, 3 series custom metrics tổng cộng, 5 alarms, budget alerts $10/$15/$18 theo scope v2 khi notification được cấu hình. Budgets có độ trễ, không auto destroy/disable website; đo cost attribution nếu dùng lại tài nguyên chung.
- [ ] Migration code/owner/expiry/tombstone, identity mapping và legacy click baseline có asOf; không phục hồi geo/raw quá khứ từ counter. Canary read-only hoặc freeze/delta sync trước mở writes; rollback v2 phải hiểu schema mới.
- [ ] Giữ AWS v1 nguyên trạng tới khi có yêu cầu riêng. Sau cutover kiểm tổng chi phí account gồm cả v1; không gọi budget v2 là tổng hóa đơn dưới $20.

**Gate:** triển khai đúng kiến trúc đã kiểm chứng và nằm trong envelope ngân sách đo được; plan này không tự cấp quyền phát hành, đổi DNS hay ngừng tài nguyên cũ.
