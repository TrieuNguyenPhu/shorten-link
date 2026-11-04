# PROJECT_STATUS.md — Trạng Thái Hiện Tại Của Dự Án

> **Cập nhật lần cuối:** 2026-11-04 (**Phase 2 in_progress; Task 2.1 complete, chờ review/merge**)
> **Quy tắc bắt buộc:** File này luôn được cập nhật ngay sau khi hoàn thành mỗi Task trước khi mở Pull Request tương ứng.

---

## 1. Tổng Quan Trạng Thái Hiện Tại

| Hạng mục | Trạng thái | Chi tiết |
| --- | --- | --- |
| **Giai đoạn hiện tại (Current Phase)** | **Phase 2 — `in_progress`** | Task 2.1 đã triển khai/kiểm thử; Task 2.2 pending |
| **Task hiện tại** | **Task 2.1 — `complete`** | Bốn HTTP Lambda, DynamoDB/HTTP E2E local, race/vet/scan và bốn ZIP arm64 PASS; chờ review/merge |
| **Task tiếp theo (Next Up)** | **Task 2.2 (Phase 2)** | Terraform bootstrap và core modules sau khi Task 2.1 được review/merge |
| **Trạng thái AWS Cloud** | **Đã dọn theo xác nhận người dùng** | Người dùng xác nhận ngày 2026-10-26 đã xóa toàn bộ tài nguyên triển khai trước đó; phiên hoàn thiện Phase 0 không kiểm kê AWS hay xác nhận hóa đơn |
| **13 Bộ Skills & `AGENTS.md`** | **Đã kiểm tra file/config** | Đủ 13 file skill đại diện; 314 thư mục cấp đầu có `SKILL.md`. Có cấu hình trỏ `~/.agents/skills`; không suy ra mọi host đều tự nạp |

---

## 2. Bảng Theo Dõi Tiến Độ Từng Phase & Pull Request

| Phase | Task | Nội dung công việc | Nhánh (Branch) | Trạng thái | Pull Request |
| --- | --- | --- | --- | --- | --- |
| **Phase 0** | **Task 0.1** | Xác minh setup skills/`AGENTS.md`, hoàn thiện Reference Audit, Architecture & Master Plan | `chore/phase-00-reset-and-architecture-plan` | **complete (đã merge main qua PR #65)** | [PR #65](https://github.com/TrieuNguyenPhu/shorten-link/pull/65) |
| **Phase 1** | **Task 1.1** | Monorepo, `Makefile`, Go module, Domain Models, Base62 Generator & URL Validator + Unit Tests, CI nền | `fix/phase-01-task-01-final-review` | complete | [PR #68 — đã merge main](https://github.com/TrieuNguyenPhu/shorten-link/pull/68); gốc #66 |
| **Phase 1** | **Task 1.2** | Observability Middleware (`slog` JSON, CloudWatch EMF, X-Ray) & Script đóng gói Lambda `arm64` ZIP | `fix/phase-01-task-02-main-integration` | complete | [PR #69 — đã merge main](https://github.com/TrieuNguyenPhu/shorten-link/pull/69); gốc #67 |
| **Phase 2** | **Task 2.1** | DynamoDB Repository, Link Service, EventBridge publisher, local HTTP harness & 4 HTTP Lambdas + Tests | `feat/phase-02-task-01-core-crud-lambdas` | complete (chờ review/merge) | [PR #71](https://github.com/TrieuNguyenPhu/shorten-link/pull/71) |
| **Phase 2** | **Task 2.2** | Terraform Bootstrap (S3 state/lockfile, OIDC) & Core Modules, Cognito Google-only, custom event bus | `feat/phase-02-task-02-terraform-core-infra` | ⏳ Pending | — |
| **Phase 3** | **Task 3.1** | Go Lambda `click-aggregator` (`TransactWriteItems`, `CancellationReasons`, `ReportBatchItemFailures`) + Tests | `feat/phase-03-task-01-click-aggregator-lambda` | ⏳ Pending | — |
| **Phase 3** | **Task 3.2** | Terraform Module `event_pipeline` (nhận bus từ Task 2.2, tạo rule, SQS `ClickQueue` + `ClickDLQ`, ESM) | `feat/phase-03-task-02-terraform-event-pipeline` | ⏳ Pending | — |
| **Phase 4** | **Task 4.1** | Go Lambdas `ws-connect`, `ws-disconnect`, `PostToConnection` Pusher (`410 Gone`) & `moderate-check` + ASL | `feat/phase-04-task-01-websocket-and-moderation-lambdas` | ⏳ Pending | — |
| **Phase 4** | **Task 4.2** | Terraform Modules `websocket_api` (`$connect`, `$disconnect`, stage `prod`) & `moderation_sfn` | `feat/phase-04-task-02-terraform-websocket-and-sfn` | ⏳ Pending | — |
| **Phase 5** | **Task 5.1** | Next.js static, Google-only qua Cognito PKCE, Landing/Login/Callback theo Doodle & Sketch UI | `feat/phase-05-task-01-nextjs-auth-and-pages` | ⏳ Pending (auth/UI); scaffold đã kiểm | [PR #70 — scaffold/static export](https://github.com/TrieuNguyenPhu/shorten-link/pull/70) |
| **Phase 5** | **Task 5.2** | Dashboard & My Links Doodle & Sketch, API thật & Native WebSocket Hook (`useRealtimeClicks`) | `feat/phase-05-task-02-dashboard-api-and-websocket` | ⏳ Pending | — |
| **Phase 5** | **Task 5.3** | Terraform Module `frontend_hosting` (Private S3 + CloudFront OAC + Cache Policies + URL Rewrite) | `feat/phase-05-task-03-terraform-s3-cloudfront-oac` | ⏳ Pending | — |
| **Phase 6** | **Task 6.1** | Terraform Modules `observability` (Alarms, SNS, Dashboard, X-Ray) & `codedeploy_canary` (`live` alias) | `feat/phase-06-task-01-observability-and-canary` | ⏳ Pending | — |
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
