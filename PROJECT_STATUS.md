# PROJECT_STATUS.md — Trạng Thái Hiện Tại Của Dự Án

## Phase 6 — 2026-11-05

Task 6.1 — **complete (implementation/local)**: observability và CodeDeploy canary; Task 6.2 tiếp theo. Worktree riêng từ main `f5112f1`; không apply AWS.


> **Cập nhật lần cuối:** 2026-11-05 (UI tối giản, review toàn project và production rollout).
> **Quy tắc bắt buộc:** File này luôn được cập nhật ngay sau khi hoàn thành mỗi Task trước khi mở Pull Request tương ứng.

---

## 1. Tổng Quan Trạng Thái Hiện Tại

- **Đồng bộ PR #88 sau squash #87 — complete (local):** main `2b6ed24` có cùng tree với head #87 đã kế thừa. Giữ nguyên tree #88 khi giải quyết conflict, bao gồm callback về `/`, PKCE và dashboard lifecycle.

- **Đồng bộ PR #88 — complete (local):** kế thừa bản sửa conflict #87 và main sau #86; giữ UI đã deploy, callback về `/` và publisher chạy test trước upload.

- **Task 5.5 — UI refinement và project review — complete (implementation/deployed)** — [PR #88](https://github.com/TrieuNguyenPhu/shorten-link/pull/88): đăng nhập về `/`; Navbar/Footer ở root layout; menu Trang chủ/Dashboard khi đăng nhập/Kiến trúc dự án có active/hover; login có logo và tiêu đề căn giữa; Lobster self-hosted; giảm copy. Sửa refresh gây nhấp nháy, giữ click mới trước snapshot cũ, chặn snapshot khôi phục link đã xóa, hiển thị/copy kết quả tạo link. Frontend 27 tests và lint/typecheck/build/audit PASS; [CI #209](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41974976600) PASS cả 4 jobs.
- **Rà soát hệ thống:** Go 1.26.10 race/vet/build 8 ZIP, DynamoDB Local repository + HTTP/WebSocket E2E, govulncheck, 38 Terraform plan/mock tests (19 bootstrap + 19 dev), CloudFront rewrite và PowerShell offline PASS. Smoke AWS thật xác nhận HTTP JWT negatives; operator Lambda CRUD/owner isolation; 301 → EventBridge/SQS → counter → WebSocket; disconnect cleanup; moderation keyword. Chưa đăng nhập bằng tài khoản Google thật; browser đã tới `accounts.google.com`.

- **Đồng bộ PR #87 — complete (local):** merge `main` sau PR #86 (`9fb63e1`); giữ cả lịch sử publish và PKCE. Không đổi implementation auth.

- **OAuth callback state mismatch — `complete (implementation/deployed)` — [PR #87](https://github.com/TrieuNguyenPhu/shorten-link/pull/87):** sửa Navbar token check xóa PKCE và callback effect replay; 24 tests/lint/typecheck/build/audit production PASS. Đã publish lên CloudFront, invalidation `I32HVUHF141OJMAXY9B68R6FDG` Completed. Browser production với state đúng đi đến token endpoint (code chẩn đoán bị Cognito từ chối đúng); chưa kiểm chứng tài khoản Google thật. Nhánh `fix/phase-05-preserve-pkce-callback` từ `origin/main`.

- **Publish frontend — `complete` — [PR #86](https://github.com/TrieuNguyenPhu/shorten-link/pull/86):** đã xuất bản PR #85 (`2253a3c`) lên [website đang chạy](https://d2aamyg2mt28wl.cloudfront.net). CloudFront `EP4W63KC4Y7HZ`, invalidation `I7A20EUFG6A2VDMZ6BOQAM2I67` đã `Completed`. Build production/18 tests, hosting smoke, title/favicon năm route và đối chiếu HTML/icon với artifact đều PASS. Browser production mở được Google qua Cognito; chưa kiểm chứng sau bước nhập tài khoản Google.

- **Google login và metadata — `complete (implementation/local)` — [PR #85](https://github.com/TrieuNguyenPhu/shorten-link/pull/85):** local đã có public env từ Terraform outputs, nút Google mở được `accounts.google.com` qua Cognito; giữ stage WebSocket `/prod`, bỏ chú thích đăng nhập, favicon dùng logo link vàng và title riêng cho từng trang. Nhánh `fix/phase-05-google-login-and-page-metadata` từ `origin/main`.
- Kiểm tra phiên sửa: `pnpm lint`, `pnpm typecheck`, `pnpm build`, `pnpm test` (18/18), `git diff --check` PASS. Google client/secret trong JSON khớp IdP AWS; `.env.local` chỉ chứa public config và bị Git bỏ qua. Chưa xác minh sau bước nhập tài khoản Google; chưa publish CloudFront trong phiên này.

| Hạng mục | Trạng thái | Chi tiết |
| --- | --- | --- |
| **Giai đoạn hiện tại (Current Phase)** | **Phase 5 — complete (implementation/deployed)** | Frontend tĩnh và hosting S3/CloudFront; Google account E2E chưa thực hiện |
| **Task hiện tại** | **Task 5.5 — complete (implementation/deployed)** | UI refinement, dashboard lifecycle và project review |
| **Task tiếp theo (Next Up)** | **Task 6.1** | Observability nâng cao và CodeDeploy canary |
| **Trạng thái AWS Cloud** | **Core/pipeline/realtime/hosting đang phục vụ** | Account 654822777701, ap-southeast-1; phiên này publish frontend, không apply Terraform |
| **13 Bộ Skills & `AGENTS.md`** | **Đã kiểm tra file/config** | Đủ 13 file skill đại diện; 314 thư mục cấp đầu có `SKILL.md`. Có cấu hình trỏ `~/.agents/skills`; không suy ra mọi host đều tự nạp |

---

## 2. Bảng Theo Dõi Tiến Độ Từng Phase & Pull Request

| Phase | Task | Nội dung công việc | Nhánh (Branch) | Trạng thái | Pull Request |
| --- | --- | --- | --- | --- | --- |
| **Phase 0** | **Task 0.1** | Xác minh setup skills/`AGENTS.md`, hoàn thiện Reference Audit, Architecture & Master Plan | `chore/phase-00-reset-and-architecture-plan` | **complete (đã merge main qua PR #65)** | [PR #65](https://github.com/TrieuNguyenPhu/shorten-link/pull/65) |
| **Phase 1** | **Task 1.1** | Monorepo, `Makefile`, Go module, Domain Models, Base62 Generator & URL Validator + Unit Tests, CI nền | `fix/phase-01-task-01-final-review` | complete | [PR #68 — đã merge main](https://github.com/TrieuNguyenPhu/shorten-link/pull/68); gốc #66 |
| **Phase 1** | **Task 1.2** | Observability Middleware (`slog` JSON, CloudWatch EMF, X-Ray) & Script đóng gói Lambda `arm64` ZIP | `fix/phase-01-task-02-main-integration` | complete | [PR #69 — đã merge main](https://github.com/TrieuNguyenPhu/shorten-link/pull/69); gốc #67 |
| **Phase 2** | **Task 2.1** | DynamoDB Repository, Link Service, EventBridge publisher, local HTTP harness & 4 HTTP Lambdas + Tests | `feat/phase-02-task-01-core-crud-lambdas` | complete (đã merge main) | [PR #71](https://github.com/TrieuNguyenPhu/shorten-link/pull/71) |
| **Phase 2** | **Task 2.2** | Terraform Bootstrap (S3 state/lockfile, OIDC) & Core Modules, Cognito Google-only, custom event bus | `feat/phase-02-task-02-terraform-core-infra` | complete (đã merge main) | [PR #72](https://github.com/TrieuNguyenPhu/shorten-link/pull/72) |
| **Phase 3** | **Task 3.1** | Go Lambda `click-aggregator` (`TransactWriteItems`, `CancellationReasons`, `ReportBatchItemFailures`) + Tests | `feat/phase-03-task-01-click-aggregator-lambda` | complete (đã merge main) | [PR #74](https://github.com/TrieuNguyenPhu/shorten-link/pull/74) | |
| **Phase 3** | **Task 3.2** | Terraform Module `event_pipeline` (nhận bus từ Task 2.2, tạo rule, SQS `ClickQueue` + `ClickDLQ`, ESM) | `feat/phase-03-task-02-terraform-event-pipeline` | complete (đã merge main6dee1ca) | [PR #75](https://github.com/TrieuNguyenPhu/shorten-link/pull/75) | |
| **Phase 4** | **Task 4.1** | Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher (`410 Gone`) & `moderate-check` + ASL | `feat/phase-04-task-01-websocket-and-moderation-lambdas` | complete (đã merge mainf6e3769) | [PR #76](https://github.com/TrieuNguyenPhu/shorten-link/pull/76) |
| **Phase 4** | **Task 4.2** | Terraform Modules `websocket_api` (`$connect`, `$disconnect`, stage `prod`) & `moderation_sfn` | `feat/phase-04-task-02-terraform-websocket-and-sfn` | complete (đã merge main501c049; review fix riêng) | [PR #77](https://github.com/TrieuNguyenPhu/shorten-link/pull/77); [fix #78](https://github.com/TrieuNguyenPhu/shorten-link/pull/78) |
| **Phase 5** | **Task 5.1** | Next.js static, Google-only qua Cognito PKCE, Landing/Login/Callback theo Doodle & Sketch UI | `feat/phase-05-task-01-nextjs-auth-and-pages` | complete (implementation/local) | [PR #79](https://github.com/TrieuNguyenPhu/shorten-link/pull/79), [PR #70 — scaffold/static export](https://github.com/TrieuNguyenPhu/shorten-link/pull/70) |
| **Phase 5** | **Task 5.2** | Dashboard & My Links Doodle & Sketch, API thật & Native WebSocket Hook (`useRealtimeClicks`) | `feat/phase-05-task-02-dashboard-api-and-websocket` | complete (implementation/deployed) | [PR #82](https://github.com/TrieuNguyenPhu/shorten-link/pull/82) và Task 5.5 |
| **Phase 5** | **Task 5.3** | Terraform Module `frontend_hosting` (Private S3 + CloudFront OAC + Cache Policies + URL Rewrite) | `feat/phase-05-task-03-terraform-s3-cloudfront-oac-v2` | complete (implementation/local) | [PR #81](https://github.com/TrieuNguyenPhu/shorten-link/pull/81) |
| **Phase 5** | **Task 5.5** | UI tối giản, font Lobster, lifecycle dashboard và project review/release | `fix/phase-05-ui-review-production` | complete (implementation/deployed) | [PR #88](https://github.com/TrieuNguyenPhu/shorten-link/pull/88) |
| **Phase 6** | **Task 6.1** | Terraform Modules `observability` (Alarms, SNS, Dashboard, X-Ray) & `codedeploy_canary` (`live` alias) | `feat/phase-06-task-01-observability-and-canary-v2` | complete (implementation/local) | [PR #89](https://github.com/TrieuNguyenPhu/shorten-link/pull/89) |
| **Phase 6** | **Task 6.2** | GitHub Actions CI/CD (AWS OIDC, Canary Deploy, S3/CloudFront Sync), k6 Load Test & Runbook E2E | `feat/phase-06-task-02-cicd-k6-and-runbook` | ⏳ Pending | — |

---

## 3. Tài Liệu Cốt Lõi Đã Khởi Tạo Ở Phase 0

- [`AGENTS.md`](./AGENTS.md): Quy chuẩn làm việc, kiến trúc bắt buộc, tích hợp 13 bộ skills và quy trình PR theo từng task.
- [`docs/phase-0-reference-audit.md`](./docs/phase-0-reference-audit.md): Phân tích chi tiết 21 bài viết và toàn bộ mã nguồn repo tham chiếu `nghiadaulau/serverless-url-shortener-aws`.
- [`docs/architecture-and-contracts.md`](./docs/architecture-and-contracts.md): Sơ đồ kiến trúc Mermaid, hợp đồng API HTTP/WebSocket, thiết kế DynamoDB Single-Table & giải thích 9 access patterns.
- [`docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`](./docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md): Kế hoạch triển khai chi tiết từng Phase (Phase 0 → Phase 6) và từng Task.
- [`task_plan.md`](./task_plan.md), [`findings.md`](./findings.md), [`progress.md`](./progress.md): Hệ thống file quản lý bộ nhớ làm việc theo chuẩn `planning-with-files`.

## 4. Bằng Chứng Kiểm Tra Và Giới Hạn

- Tài liệu đã đối chiếu source ở commit `a5ffed9780d0447fc75a42def3693732bdc87907` (21 tracked files) và đủ 21 bài qua cả hai trang series; các giới hạn nguồn được ghi trong audit.
- Kiểm tra Phase 0: `git diff --check`, liên kết Markdown nội bộ, JSON examples, số bài và file nguồn. Không chạy `go test`, `pnpm build` hay `terraform validate` vì các dự án thực thi chỉ bắt đầu từ Phase 1–2.
- E2E Google → Cognito → CloudFront/API, resource inventory và chi phí AWS chưa xác minh; Google OAuth Web client/secret là đầu vào bên ngoài cần chuẩn bị ở Task 2.2, không có secret trong repository.
- Frontend từng untracked ở Phase 0, được commit tại `4a5c28a` và nay tách khỏi PR backend để tích hợp scaffold có static export/CI. Đã kiểm tra lint/typecheck/build và artifact HTML/assets; Google auth, Doodle UI và dashboard vẫn chưa triển khai (Task 5.1/5.2).

## 5. Bằng chứng lịch sử 2026-10-27 / Task 1.1
- Go 1.26.5: test/vet trên Windows PASS; `GOTOOLCHAIN=go1.26.5 go -C backend test -race -v ./...` và vet trên WSL Ubuntu/GCC PASS.
- `gofmt -l backend` không có file; `git diff --check` PASS. Review độc lập domain/config/CI không có finding cần sửa.
- Test-first đã xác nhận RED do các API chưa tồn tại, sau đó GREEN. Coverage config 100%, domain 88.2%; không suy ra bảo đảm nghiệp vụ chưa triển khai.
- PR phụ thuộc #65; frontend untracked được giữ nguyên. Chưa gọi AWS hoặc chạy Terraform/pnpm.
- Cập nhật review Task1.1: minimum Go/CI chuyển sang1.26.10 để khắc phục stdlib vulnerabilities phát hiện trong Phase1; bằng chứng1.26.5 bên trên là lịch sử.

## 6. Bằng chứng lịch sử 2026-10-27 / Task 1.2
- Minimum Go/CI **1.26.10**; code ứng dụng SDK v2, aws-lambda-go v1.55.1, X-Ray SDK v1.8.5 và go.sum được pin.
- Windows: `go -C backend test ./...`, `go -C backend vet ./...` PASS; PS wrapper chạy từ thư mục khác có dấu cách và trả lỗi đúng khi compiler fail.
- WSL Ubuntu/GCC: `GOTOOLCHAIN=go1.26.10 go -C backend test -race ./...` và vet PASS cả 4 packages; Bash wrapper được kiểm trực tiếp trong integration test.
- Packaging test build chương trình Go thật trong thư mục tạm, kiểm ELF64 AArch64 không dynamic linker, ZIP chỉ có bootstrap ở gốc/quyền0755 và giữ artifact cũ khi build lỗi. Chưa có cmd nghiệp vụ ở Phase1 nên lệnh build repo trả lỗi rõ ràng, không phát ZIP handler giả.
- `govulncheck@v1.9.0 -show verbose ./...` trên Go1.26.10: exit0, 0 symbol/package vulnerabilities; còn 2 module-only GO-2022-0646/GO-2022-0635 ở S3 Crypto của SDKv1 gián tiếp, không import/call bởi app. Không suppress. CI bổ sung reachable vulnerability scan.
- `go mod verify`, `gofmt -l backend`, `git diff --check` PASS. Review độc lập cả hai task không có finding code phải sửa.
- Ruling: EmitCountMetric trả error để không nuốt lỗi JSON/write; WithRequestID lấy invocation context cho logger. Hai wrapper dùng Go helper chung để bảo toàn quyền Unix trên Windows.
- X-Ray SDK maintenance và SDKv1 gián tiếp là giới hạn được công bố; chưa xác minh trace export trên AWS. Không gọi AWS, Terraform hoặc load test; frontend untracked được giữ nguyên.
- Thứ tự review/merge: #65 → #66 → #67. Retarget về main sau khi PR phụ thuộc đã merge; không tự merge.


## 7. Sửa final review Task 1.1 — 2026-11-04
- PR #65 đã squash vào main tại `c99bd5e`; #66 đã merge vào nhánh Phase 0 sau đó, nên main chưa nhận Task 1.1. Nhánh sửa mới tạo từ origin/main, phục hồi đúng ba commit Task 1.1; không gộp Task 1.2/frontend.
- RED: regression test từ chối `127.0.0.999`, `256.1.2.3`, IPv4 sai có trailing dot và năm octet thất bại trước sửa. GREEN sau khi không cho địa chỉ IPv4 dạng số sai rơi vào nhánh DNS.
- Linux / Go 1.26.10: `go -C backend test -race -count=1 -cover ./...`, `go -C backend vet ./...`, `go -C backend mod verify`, `gofmt -l backend` và `git diff --check` PASS; config coverage 100%, domain 88.9%.
- Đã xác nhận kết nối GitHub/quyền ghi. PR #68/#69 đã đưa đầy đủ Task 1.1/1.2 lên main; Phase 2 chưa bắt đầu.

- PR tích hợp Task 1.1 vào main: [#68](https://github.com/TrieuNguyenPhu/shorten-link/pull/68), đã qua CI và merge main tại `4725d97`.

## 8. Tích hợp Task 1.2 — 2026-11-04
- PR #68 đã merge main tại `4725d97`; nhánh `fix/phase-01-task-02-main-integration` tạo từ main này, chỉ phục hồi Task 1.2 và tài liệu liên quan.
- Go 1.26.10/Linux: test-race-count1-cover/vet/mod-verify/gofmt/diff-check PASS, cả 4 packages. Packaging test chạy compiler và Bash wrapper thật, kiểm ELF AArch64/tĩnh, ZIP root bootstrap0755 và bảo toàn artifact khi build lỗi.
- govulncheck v1.9.0 verbose: exit0, 0 symbol/package vulnerabilities; 4 module-only: GO-2026-6866/GO-2026-6860 ở x/net và GO-2022-0646/GO-2022-0635 ở S3 Crypto SDKv1, không có đường gọi từ ứng dụng. Không suppress hoặc tuyên bố không còn mọi cảnh báo.
- Chưa có handler nghiệp vụ; không chạy AWS/Terraform. Static export sửa qua PR riêng trước Phase 2.

- PR tích hợp Task 1.2 vào main: [#69](https://github.com/TrieuNguyenPhu/shorten-link/pull/69), CI Linux/Windows PASS, đã merge main tại `4648f93`.

## 9. Sửa static export và chốt final review — 2026-11-04
- Scaffold frontend được tích hợp riêng từ main sau #69, không gộp với PR backend; đây chưa phải hoàn thành Task 5.1 (Google login/UI vẫn pending).
- Đã đọc guide static export đi kèm next16.4.3 trước khi sửa. Cấu hình `output: "export"`, `trailingSlash: true`, `images.unoptimized: true`; bỏ `next start`, chuẩn hóa LF ở package/workspace YAML.
- Regression RED trên scaffold cũ: build exit0 nhưng thiếu out/index.html. GREEN sau sửa: corepack pnpm@10.33.2 install --frozen-lockfile/lint/typecheck/build/test/audit --prod PASS trên Node26.13.0. Hai test kiểm HTML trang chủ/404, CSS/JS/images tồn tại và không cần image server; out/index.html và out/404.html tồn tại.
- CI thêm frontend lint/typecheck/static-build/artifact-tests/audit, dùng pnpm10.33.2/Node26.13.0. Backend/race/Windows packaging và vulnerability scan vẫn giữ nguyên.
- Phase 1 hoàn tất nền tảng local; kế tiếp Task 2.1. Chưa triển khai AWS, Terraform, JWT authorizer, E2E nghiệp vụ hoặc Google login.

- PR sửa scaffold/static export: [#70](https://github.com/TrieuNguyenPhu/shorten-link/pull/70), base main sau #69; trạng thái CI/merge xem tại PR.


## 10. Phase 2 / Task 2.1 — 2026-11-04
- Nhánh riêng từ origin/main c9d0c85 sau PR #70; có create/resolve/list/delete Lambda thật, repository DynamoDB, service, EventBridge publisher và local HTTP harness. Không có memory repository cho luồng link.
- TDD RED do API mới chưa tồn tại; GREEN unit/race với collision tối đa 5 writes, owner lấy từ authorizer sub, conditional delete, pagination và body validation. Publisher chỉ ack EventId hợp lệ, retry entry tối đa 3 lần, deadline2s/NopRetryer; toàn request4s, metric lỗi không gây publish lại.
- Go1.26.10/Linux: `go -C backend test -race -count=1 -cover ./...`, `go -C backend vet ./...`, `go -C backend mod verify` PASS. Các regression bổ sung events/httpapi và runtime/harness được chạy lại với race, exit0.
- DynamoDB Local3.3.0: `DYNAMODB_ENDPOINT=http://127.0.0.1:8000 go -C backend test -race -tags=integration -count=1 -v ./internal/repository/... ./tools/local-http/...` PASS. Pagination thật vượt1MB, timestamp trong cùng giây, chống IDOR và giữ STAT/CLICK khi delete. HTTP E2E và CLI smoke bốn route PASS; EventBridge recorder là fixture tại ranh giới HTTP, không phải service AWS.
- `bash scripts/build-lambdas.sh` PASS: bốn ZIP trong backend/dist (ignored), mỗi ZIP chỉ bootstrap0755, ELF64 AArch64 không PT_INTERP. CI thêm DynamoDB Local integration/E2E và build4ZIP; fmt/diff/liên kết/JSON được kiểm trước commit.
- govulncheck v1.9.0 verbose exit0: 0 symbol/package vulnerabilities; vẫn 4 module-only advisories như mục8, không suppress. AWS SDK v2 DynamoDB/EventBridge/config/credentials đã pin trong go.mod/go.sum.
- Rà soát diff không secret/state/tfvars thực, không sửa frontend, không có transaction aggregator trong scope. Không chạy AWS/apply/load test; JWT/IAM/EventBridge pipeline và trace export giữ pending. Hướng dẫn tại [docs/local-testing.md](./docs/local-testing.md).

- PR Task 2.1: [#71](https://github.com/TrieuNguyenPhu/shorten-link/pull/71), base main, CI PASS, chờ review; chưa merge.
- CI [run #164](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41759270780) trên commit39fdbe8 PASS cả backend (race/vet/DynamoDB integration/HTTP E2E/build4ZIP/scan), packaging-windows và frontend (lint/typecheck/static build/tests/audit). Commit sau chỉ cập nhật bằng chứng tài liệu.


## 11. Phase 2 / Task 2.2 — 2026-11-04
- #71 đã được người dùng review/merge, origin/main158b127 làm baseline. Nhánh riêng feat/phase-02-task-02-terraform-core-infra; từ đây commit/PR title/description dùng tiếng Anh theo yêu cầu mới, đã lưu trong AGENTS.
- Bootstrap: state S3 private/versioning/AES256/TLS, native lockfile backend, GitHub OIDC exact aud/sub, deployment role và runtime permissions boundary. App role chỉ state key/lock prefix và app resources; không sửa bootstrap/trust/boundary. Bootstrap state mặc định local độc lập, phải bảo vệ/backup khi được phép deploy.
- Core modules: DynamoDB PAY_PER_REQUEST PK/SK/GSI1/ttl/PITR, Cognito chỉ Google/public code client, bốn Lambda provided.al2023 arm64 bootstrap128MB/5s/Active tracing, Logs14d, custom bus, HTTP API JWT scopes/CORS/throttle/default stage. API invoke qua alias live; resolve alias nhường version/routing cho CodeDeploy.
- Terraform1.17.1/AWS provider6.71.0 + lockfiles: fmt-check, init-backend=false-lockfile-readonly, validate và test hai root PASS. 8 bootstrap +8 core runs đều plan/mock, không AWS apply. Go1.26.10 race/vet và build4ZIP PASS.
- PowerShell7.5/container: tests/e2e/aws-flow.test.ps1 parser + Core/OIDC endpoint/role/missing-token preflight PASS; không gửi request AWS. aws-flow.ps1 Core/OIDC sẵn sàng cho hai Google identities/access tokens và ID/Google negative tokens khi được phép; E2E AWS chưa chạy.
- Host thiếu make, đã chạy trực tiếp toàn bộ command tương đương, không tuyên bố make check thành công. IAM Policy Autopilot không chạy vì uvx/pip/ensurepip thiếu; fallback SAR xác minh104 action names và resource capabilities. Không upload policy hoặc gọi account AWS.
- Giới hạn cần xử lý khi triển khai: chuẩn bị Google Web client/secret, bảo vệ remote state/plan, exact protected environment/main trust. API Gateway v2 child/tag endpoints scope ID/path, không dựa inheritance V1; pin http_api_ids output vào bootstrap sau first approved deployment và trước CD. DescribeUserPoolDomain cần wildcard read theo RequestedRegion. Chi tiết [runbook Terraform](./docs/terraform-core-infrastructure.md).
- CI thêm Terraform fmt/init/validate/16 plan tests và PowerShell parser/preflight; setup-terraform action pin SHA. Rà soát diff/IAM/boundary/alias/secret state và liên kết trước commit; chưa có AWS JWT/IAM/STS/X-Ray/pipeline evidence. Phase2 complete về implementation, kế tiếp Task3.1 sau review/merge Task2.2.

- PR riêng Task2.2: [#72](https://github.com/TrieuNguyenPhu/shorten-link/pull/72), base main sau #71, title/body và commits tiếng Anh. Trạng thái CI hiện tại xem tại PR; chưa merge.


## 12. Sửa final review và triển khai Phase 2 — 2026-11-04
- Người dùng đã yêu cầu sửa bốn phát hiện, review cuối rồi terraform apply. Đây là authorization triển khai mới; đã apply bootstrap/core và pin policy sau final review. #71/#72 đã merge main634ee88; nhánh fix/phase-02-final-review từ main.

- Đã sửa bốn finding: Cognito maintenance pin pool ARN và cấm retag ownership, thêm GetFunctionCodeSigningConfig; cursor pagination được người dùng chốt và cập nhật đặc tả; bounded deletion wait. Go race/vet/build4ZIP, DynamoDB Local integration/HTTP E2E, Terraform validate/fmt/18 plan tests và PowerShell preflight/deletion tests PASS. Final diff review không còn finding cần sửa; đang triển khai AWS được người dùng yêu cầu. Google Web client JSON đã được đọc trực tiếp từ file local, không commit/log secret.

- PR sửa review: [#73](https://github.com/TrieuNguyenPhu/shorten-link/pull/73); commit code b62811f. CI [run170](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41767763287) PASS backend, terraform, packaging-windows và frontend. Commit tiếp theo chốt evidence và Cognito read-back defaults.
- AWS account654822777701 / ap-southeast-1: bootstrap apply10added/0changed/0destroyed; core42added/0changed/0destroyed; pinning0added/1changed/0destroyed. Cognito pool ap-southeast-1_SGj2otrc0, API txnaexyh3d; deployment policy chỉ cho maintenance đúng pool/API, IAM Access Analyzer không có errors/security warnings trên policy thực đã pin.
- API: https://txnaexyh3d.execute-api.ap-southeast-1.amazonaws.com ; table url-shortener (PITR/TTL/deletion protection); bus url-shortener-events; bốn Lambda arm64 có alias live. Cognito Google client2dvtankl8jhb76q0qukiuhmtlg, domain https://url-shortener-dev-654822777701.auth.ap-southeast-1.amazoncognito.com . Callback hiện http://localhost:3000/auth/callback/; frontend auth/hosting vẫn Phase5.
- Smoke AWS PASS: ba private routes thiếu JWT401, malformed JWT401, public missing404; operator invoke các Lambda tạo2link, list hai trang, reject foreign delete404/cross-owner cursor400/limit101400; HTTP resolve301 qua live alias và EventBridge publisher nhận ack. Cleanup cả2link rồi HTTP404; Cognito code+PKCE authorization redirect302 về accounts.google.com. Operator invocation với authorizer context không thay thế Google access-token E2E.
- Lượt plan sau apply phát hiện Cognito thêm username/sub và6 provider metadata keys. Sửa lifecycle chỉ ignore các AWS defaults, giữ credentials/scopes/email/name được quản lý. RED actual plan exit2; GREEN sau sửa: fmt/validate/8core tests PASS và bootstrap/core detailed-exitcode đều0 (no changes). Không thay secret hoặc recreate pool.
- Bootstrap state local và backup /home/trieunguyen/.local/state/shorten-link/bootstrap-terraform.tfstate quyền0600; app state S3 private/versioned/AES256/TLS/native lockfile. OAuth JSON/tfvars/plan/state không tracked hoặc hiển thị secret. Google browser sign-in/API E2E hai tài khoản, STS OIDC trust matrix, trace export và pipeline delivery vẫn pending theo task sở hữu; không chạy load test.


## 13. Phase 3 / Task 3.1 — 2026-11-04
- Nhánh riêng từ origin/main e1e8bed sau PR#73 đã merge. Triển khai Lambda `click-aggregator`, DynamoDB transaction META/STAT/marker, classifier3CancellationReasons, consistent read xác nhận delete và SQS partial batch response. Notifier là dependency sau commit; runtime chưa gắn WebSocket tới Phase4.
- TDD RED do APIs transaction/service/runtime chưa tồn tại; GREEN unit tests cho18cancellation cases, malformed envelopes, batch10chỉ2failure, notification/metric isolation và deadline. Integration thực thi engine DynamoDB Local3.3.0 (digest đã pin), không dùng memory repository.
- Go1.26.10/Linux: `go -C backend test -race -count=1 -cover ./...`, `go -C backend vet ./...`, `go -C backend mod verify` PASS. `DYNAMODB_ENDPOINT=http://127.0.0.1:8000 go -C backend test -race -tags=integration -count=1 -v ./internal/repository/... ./tools/local-http/...` PASS; kiểm transactions/TTL/replay/concurrent count/delete race và HTTP→publisher fixture→aggregator→counter. Test contention/delete race lặp10lượt PASS.
- `bash scripts/build-lambdas.sh` PASS nămZIP; từng archive bootstrap root0755/ELF64 AArch64/no PT_INTERP. CI tự chạy tests mới/build5ZIP; đổi tên bước cho đúng phạm vi. govulncheck1.9.0 verbose exit0:0symbol/package vulnerabilities,4module-only advisories như baseline, không suppress.
- Review độc lập không có finding cần sửa; controller rà soát diff/schema/reason bounds/deadline/postcommit failures. gofmt/diff-check,13Markdown links và6JSON examples PASS. Lượt kiểm contention thiếu endpoint ban đầu fail đúng guard, chạy lại đủ biến môi trường PASS; không coi lỗi setup là lỗi nghiệp vụ.
- Không đổi Terraform, không apply/call AWS hoặc load test; event bus/rule/SQS/ESM/DLQ/IAM và AWS delivery E2E thuộc Task3.2, WebSocket thuộc Phase4. Kế tiếp Task3.2 từ main sau Task3.1 merge; Phase3 vẫn in_progress.

- PR riêng Task3.1: [#74](https://github.com/TrieuNguyenPhu/shorten-link/pull/74), base main e1e8bed, commit fbd9ab1; chưa merge. Xuất bản bằng GitHub connector vì host thiếu Git author/gh/credential helper; remote tree ee08d52 trùng index đã kiểm thử.

- CI [run #174](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41784113801) trên commit587dc7f PASS cả bốn job: backend (race/vet/transaction integration/HTTP analytics E2E/build5ZIP/scan), Terraform, packaging-windows và frontend. Commit chốt sau chỉ sửa tài liệu/README, không đổi implementation.


## 14. Phase3 / Task3.2 — 2026-11-04
- Main113d969 sau PR#74 đã merge; nhánh riêng feat/phase-03-task-02-terraform-event-pipeline. Module nhận existing custom bus, tạo rule/envelope target/SQS standard30s/maxReceive3/DLQ14d/SSE-SQS/TLS/source policy/redrive allow và ESM batch10/partial failures/live aggregator alias. Aggregator dùng module Lambda chung; root outputs thêm queue/rule/ESM và năm role/alias.
- Bootstrap boundary bổ sung UpdateItem/poll ClickQueue và năm handler. Managed policy pipeline tách khỏi inline do regression11256>10240; IAM deployment scope exact rule/queues, ESM operator provision lần đầu rồi pin UUID + FunctionArn live, bảo vệ ownership tags. Chi tiết/ruling tại docs/terraform-event-pipeline.md; không thay tài nguyên AWS trong phiên.
- Terraform1.17.1/provider6.71.0: fmt-check/init-backend=false-lockfile-readonly/validate/test PASS cả hai root,14+13=27plan/mock tests. Go1.26.10 race/vet/build5ZIP PASS. PowerShell7.5 container --network none parser/preflight Core/OIDC/Pipeline/deletion/JSON shape regression PASS. Không có bằng chứng AWS pipeline delivery/DLQ thật.
- Review độc lập Terraform không finding; PS JSON scalar/array finding đã sửa và scoped re-review PASS. IAM Autopilot unavailable (uvx exit127); dùng SAR4services kiểm action/resource/condition, không upload policy/call account. CI mở rộng tự chạy tests mới; runbook Pipeline nêu rõ exact account/output validation, bounded polls và cleanup test-only.
- Phase3 complete về implementation/local, AWS rule/queues/aggregator/ESM chưa apply; Scenario Pipeline thật pending. Google Core E2E/OIDC trust matrix còn pending như Phase2. Task4.1 bắt đầu sau review/merge Task3.2; không tự merge hoặc triển khai AWS.

- PR riêng Task3.2: [#75](https://github.com/TrieuNguyenPhu/shorten-link/pull/75), base main113d969, implementation commit a2f926f; chưa merge. Xuất bản bằng GitHub connector, tree eed7829 trùng index đã kiểm; checkout giữ để sửa review. CI [#177](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41786916663) trên a2f926f PASS cả bốn job; commit chốt sau chỉ cập nhật tài liệu.


## 15. Phase4 / Task4.1 — 2026-11-04
- Nhánh riêng từ main6dee1ca sau PR#75 merge. Có đủ8Lambda thật, thêm SDK Management API v1.41.3; không nâng dependency khác.
- Test-first RED API missing → GREEN. Go1.26.10 `test -race -count=1 -cover ./...`, `vet ./...`, `mod verify` PASS; regression pusher thật qua aggregator (Get/Query/Post/cleanup lỗi đều ack sau commit) chạy race PASS.
- DynamoDB Local3.3.0 integration `-race -tags=integration -count=1` cả repository và tools/local-http PASS: CRUD/analytics cũ, connect/disconnect/TTL, query>1MB, push SigV4 ở stage/prod,410cleanup,403isolation và duplicate không push lại. HTTP fixtures thay thế EventBridge/Management API tại ranh giới, không chứng minh handshake/IAM/ESM AWS.
- Build8ZIP thật PASS; từng ZIP chỉ bootstrap0755, ELF64 AArch64, không interpreter. `govulncheck@v1.9.0 -show verbose` exit0,0symbol/package vulnerabilities,4module-only advisories cũ.
- ASL contract tests PASS cho Lambda output trực tiếp, Retry/Catch/Fail và status updates; chưa thực thi Step Functions. Runbook manual safe/reject/retry/Fail công bố giới hạn upsert, không nối create/redirect.
- Rà soát thủ công diff/security: owner conditions và transaction cancellation classification cũ giữ nguyên; notifier chỉ sau ClickApplied, chỉ410cleanup, không thêm secret/state/tfvars/artifact. Task4.2/Terraform và AWS execution history còn pending; không apply/load test.

- PR riêng Task4.1: [#76](https://github.com/TrieuNguyenPhu/shorten-link/pull/76), base main6dee1ca, code commit affc345; CI #180 PASS, chờ review, chưa merge. Tree c90b0cf đã đối chiếu local/remote.

- CI [run #180](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41790392763) trên affc345 PASS cả4jobs: backend (race/vet/DynamoDB Local/HTTP-WebSocket E2E/build8ZIP/scan), terraform (fmt/validate/tests/PS offline), packaging-windows và frontend (lint/typecheck/static build/artifact tests/audit). Commit chốt chỉ cập nhật tài liệu; chưa merge/apply.


## 16. Phase4 / Task4.2 — 2026-11-04
- Nhánh riêng từ mainf6e3769 sau PR#76 merge. Module websocket_api tạo native WebSocket proxy connect/disconnect, public code subscription, prod auto-deploy, permission đúng account/API/stage/route/live. Root thêm3Lambda, tổng8roles/aliases và endpoint outputs. Aggregator nhận WS_ENDPOINT và Query GSI1/Delete connection/ManageConnections POST đúng API/prod.
- Module moderation_sfn render ASL gốc, STANDARD/manual; trust SourceAccount/SourceArn exact machine, role/boundary riêng chỉ invoke moderate-check:live và UpdateItem đúng bảng. Bootstrap thêm output moderation_boundary_arn bắt buộc cho dev; pin websocket_api_ids trước khi cho push/CD.
- RED module/API missing, quota inline12314>10240 và provider-required permissions; GREEN sau implementation và tách managed Lambda/API/realtime deployment policies. Không mở quyền sửa bootstrap/StartExecution hoặc wildcard push. Provider6.71.0 cần ListStateMachineVersions và plan-time ValidateStateMachineDefinition (Resource* chỉ với RequestedRegion).
- Terraform1.17.1/provider6.71.0 fmt/init-backend=false-lockfile-readonly/validate PASS;19bootstrap+18dev=37plan/mock tests PASS. Go1.26.10 race/vet/build8ZIP PASS. PS7.5 container network=none parser/Core/OIDC/Pipeline/Realtime negative output/missing-token preflight PASS.
- Scenario Realtime đã viết cho safe/reject/4attempts-Fail/2real sockets; cleanup chờ terminal trước delete. Chưa chạy AWS scenario/execution history/handshake/IAM/delivery; chưa apply hoặc load test. Docs công bố operator-first pinning, logging Phase6 và ASL upsert limitation.
- Rà soát thủ công Terraform/IAM/PS: owner conditions/cancellation code cũ giữ nguyên, endpoint/account/resource validation và GetApi ownership trước bearer HTTP/socket. Không có secret/state/tfvars thực/ZIP trong diff. Kế tiếp Task5.1 sau review/merge Task4.2.

- PR riêng Task4.2: [#77](https://github.com/TrieuNguyenPhu/shorten-link/pull/77), base mainf6e3769, code commit db716ad; tree af3e6d7 trùng local/remote. CI #183 PASS, chờ review; chưa merge.

- CI [run #183](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41793539394) trên db716ad PASS cả4jobs: backend (race/vet/DynamoDB Local/HTTP-WebSocket E2E/build8ZIP/scan), terraform (37plan tests, validate/fmt/PS preflight), packaging-windows và frontend static checks. Commit chốt sau chỉ tài liệu; chưa merge/apply.


## 17. Sửa review Phase4 / Task4.2 — 2026-11-04
- Finding P2: StartExecution được AWS nhận nhưng mất response khiến registry rỗng, cleanup xóa link trước khi workflow kết thúc. PR #77 đã merge, sửa trên fix branch từ main501c049: đặt name/ARN và đăng ký trước request; kiểm ARN response, đối soát trạng thái và stop/poll trước cleanup. Chỉ bốn trạng thái SUCCEEDED/FAILED/TIMED_OUT/ABORTED được phép xóa; unknown/nonterminal giữ link và ARN để operator xử lý.
- Test-first RED trên code cũ → GREEN sau sửa. PowerShell7.5 network=none PASS toàn bộ suite và tám lifecycle cases: lost-running/lost-completed/unknown/still-running/pending/poll-error/lost-stop/success. Test thực thi hàm start và finally thật, chỉ mock AWS/HTTP/disposable clients/delay.
- Không sửa Go/Terraform/ASL hoặc quyền AWS. Không chạy apply/E2E AWS; CI toàn repo sẽ kiểm tra hồi quy cho revision mới.

- Khi đồng bộ phát hiện PR #77 đã merge tại main501c049; bản sửa không thể cập nhật PR đã đóng. Tạo branch fix/phase-04-task-02-execution-cleanup từ main (tree baseline trùng PR #77), giữ đúng tám file sửa; chuẩn bị PR fix riêng.

- PR fix [#78](https://github.com/TrieuNguyenPhu/shorten-link/pull/78), base main501c049 → ff45f5a, tree74817f2 đã khớp local/remote; CI #186 PASS, chờ review.

- CI [run #186](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41798012440) trên ff45f5a PASS backend, terraform (37plan tests + PS preflight/tám lifecycle cases), packaging-windows và frontend. Commit chốt chỉ tài liệu: URL PR/CI và khôi phục nguyên lịch sử progress.md; không sửa code/apply/merge.

## 18. Phase 5 / Task 5.1 — 2026-11-05

- `pnpm lint`, `pnpm typecheck`, `pnpm test` (12 tests), `pnpm build` và artifact tests đều PASS trong `frontend/`.
- Build tạo `out/index.html`, `out/login/index.html`, `out/auth/callback/index.html`, `out/dashboard/index.html`, `out/404.html` và static assets.
- Auth kiểm `state`, PKCE S256, `id_token.nonce`, expiry, URL allowlist và dọn query callback; không có password/signup flow.
- Chưa chạy Terraform apply, CloudFront hosting hoặc Google browser federation E2E trong task local này.
- Trong working tree ban đầu, full Go suite/vet bị chặn bởi thay đổi backend ngoài task: `backend/internal/config/config.go:27` giữ biến `dest` nhưng dòng gán cấu hình đã bị xóa; chưa sửa trong commit frontend.


## 19. Phase 5 / Task 5.3 — 2026-11-05

- Tạo nhánh mới từ `origin/main` tại commit `702b97a` để tránh mang theo lịch sử tài liệu gây conflict của PR #80.
- Commit code hosting: `20caa84` (`feat: add private S3 and CloudFront frontend hosting`).
- Terraform dev: `fmt -check`, `validate` và `terraform test` PASS, tổng cộng 19 test pass; CloudFront viewer rewrite Node test được đưa vào CI.
- Chưa chạy `terraform apply` trong bước tạo PR; tài nguyên AWS hiện hữu không bị thay đổi.
- Đã mở [PR #81](https://github.com/TrieuNguyenPhu/shorten-link/pull/81) từ `main` (`702b97a`); GitHub xác nhận branch mergeable, không còn conflict.

## 20. Phase 5 / Task 5.4 — 2026-11-05

- PR #82 cập nhật trải nghiệm frontend Shrnk trên nền static export: logo/icon link dùng chung, Navbar nhận biết phiên đăng nhập, liên kết Source Code, route `/architecture/`, landing tạo link khi đã đăng nhập và dashboard chỉ hiển thị analytics links.
- Loại bỏ form tạo link trùng lặp trong Dashboard/My Links, nút mũi tên và các helper caption nhỏ; giữ API thật và hook WebSocket realtime hiện có.
- Kiểm tra local trên branch PR: `pnpm lint`, `pnpm typecheck`, `pnpm build` và `pnpm test` (16 pass). Không chạy `terraform apply` hoặc load test.
- PR: [#82](https://github.com/TrieuNguyenPhu/shorten-link/pull/82), base `main`; đang đồng bộ lại với `origin/main` để xử lý merge conflict.


## 2026-11-05 — UI refinement, project review và production rollout

- **Status: complete (implementation/deployed)**. Yêu cầu: đăng nhập về trang chủ; sửa dashboard nhấp nháy; layout chung; menu active/hover; login tối giản; font Lobster; review toàn project, sửa lỗi và triển khai production.
- Nhánh `fix/phase-05-ui-review-production` tạo từ `origin/main` (`2253a3c`), kế thừa hai commit sửa PKCE #87 để giữ luồng đăng nhập an toàn. Giữ nguyên các diff có sẵn để rà soát, không tự hoàn nguyên.

Task 6.1: [PR #89](https://github.com/TrieuNguyenPhu/shorten-link/pull/89), code tree `a1706d5` đã đối chiếu local/remote; chưa merge/apply. Git HTTPS thiếu credential, xuất bản qua GitHub connector.

Task 6.1: CI [#218](https://github.com/TrieuNguyenPhu/shorten-link/actions/runs/41979476998) PASS toàn bộ bốn jobs trên commit36faa086. AWS apply/canary thật vẫn pending; Task6.2 có [PR #90](https://github.com/TrieuNguyenPhu/shorten-link/pull/90) riêng từ main.

Review Phase6.1: AWS tài liệu hóa SourceAccount/SourceArn cho CodeDeploy Lambda; sửa nhận định comment cũ, thêm exact account/group trust và regression RED→GREEN. Không đổi alias/deploy AWS. Chạy lại Terraform dev tests/validate/fmt sau sửa; CI cũ #218 là bằng chứng trước hardening, CI mới sẽ kiểm commit này.
