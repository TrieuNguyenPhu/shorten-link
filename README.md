# Serverless URL Shortener with Realtime Analytics (AWS + Go + Terraform + Next.js)

Đặc tả và kế hoạch tái triển khai hệ thống **Serverless URL Shortener with Realtime Analytics** trên AWS bằng **Golang (`provided.al2023`, `arm64`)**, **Terraform (HCL)**, và **Next.js App Router (`output: "export"`)** trên **Amazon S3 (Private) + CloudFront Origin Access Control (OAC)**.

Phase 1 cung cấp nền tảng Go qua PR #68/#69; scaffold frontend có static export và kiểm tra artifact. Phase 2 có bốn HTTP Lambda, DynamoDB repository, EventBridge publisher, HTTP harness và Terraform bootstrap/core; AWS core đã triển khai ngày 2026-11-04 qua đợt final review PR #73. Task 3.1 bổ sung click-aggregator với transaction idempotency và kiểm thử DynamoDB Local/HTTP analytics E2E; Task3.2 bổ sung Terraform EventBridge rule/SQS/DLQ/ESM cùng Scenario Pipeline; phần pipeline chưa được apply trên AWS. Phase4 bổ sung WebSocket Lambdas/pusher, moderation Lambda/ASL và Terraform WebSocket/Step Functions; implementation/local đã kiểm, AWS Phase3/4 chưa deploy. Google login frontend, Doodle UI và dashboard chưa hoàn thiện. Các phiên Phase3/4 chỉ kiểm thử local, không thay đổi tài nguyên AWS.

Dự án bám sát kiến trúc hướng sự kiện, thiết kế bảng đơn DynamoDB và luồng xử lý của:
- **Chuỗi 21 bài viết:** [Serverless Thực Chiến Trên AWS](https://kkloudtarus.net/blog/series/serverless-on-aws-event-driven)
- **Repository gốc:** [nghiadaulau/serverless-url-shortener-aws](https://github.com/nghiadaulau/serverless-url-shortener-aws)

---

## 1. Bảng Đối Chiếu Thay Đổi Kiến Trúc

| Thành phần | Bản gốc | Bản triển khai này |
| --- | --- | --- |
| **Lambda Runtime** | Node.js 22 / TypeScript (`esbuild`) | **Golang** (`provided.al2023`, `arm64`, binary `bootstrap`) |
| **Infrastructure as Code** | AWS SAM (`template.yaml`) | **Terraform (HCL)** quản lý 100% hạ tầng |
| **Frontend Dashboard** | `web/index.html` (Vanilla HTML/JS) | **Next.js App Router + TypeScript + Tailwind CSS** (`output: "export"`) |
| **Frontend Hosting** | File tĩnh cục bộ | **Amazon S3 (Private) + CloudFront (OAC)** |
| **Xác thực** | Cognito email/mật khẩu trong web gốc | **Chỉ Google qua Cognito User Pool, Authorization Code + PKCE** |
| **Phong cách UI** | HTML/JS tối giản | **Hand-Drawn / Doodle & Sketch UI**, responsive, dễ đọc |
| **Dịch vụ Backend AWS** | API Gateway (HTTP + WebSocket), DynamoDB Single-Table (`PK`/`SK` + Sparse `GSI1` + `ttl`), Cognito JWT, EventBridge, SQS + DLQ, Step Functions, CloudWatch/X-Ray, SNS, CodeDeploy Canary | **Giữ dịch vụ, schema và access patterns; sửa lỗi publish, transaction và notifier theo đặc tả** |

---

## 2. Tài Liệu & Trạng Thái Dự Án

- 📊 **Trạng thái hiện tại của dự án:** [`PROJECT_STATUS.md`](./PROJECT_STATUS.md)
- 🤖 **Quy chuẩn Agent & 13 Bộ Skills:** [`AGENTS.md`](./AGENTS.md)
- 🔍 **Báo cáo Audit bản gốc (21 bài + source code):** [`docs/phase-0-reference-audit.md`](./docs/phase-0-reference-audit.md)
- 🏗️ **Kiến trúc Mermaid, Hợp đồng API & DynamoDB Single-Table:** [`docs/architecture-and-contracts.md`](./docs/architecture-and-contracts.md)
- 🗺️ **Kế hoạch triển khai chi tiết (Phase 0 → Phase 6):** [`docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`](./docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md)
- 📝 **Bộ nhớ làm việc (`planning-with-files`):** [`task_plan.md`](./task_plan.md) · [`findings.md`](./findings.md) · [`progress.md`](./progress.md)

---

## 3. Lộ Trình Phát Triển (Development Phases)

1. **Phase 0 — Reference Audit & Architecture** *(Hoàn tất)*
2. **Phase 1 — Monorepo & Local Foundation** *(Task 1.1 & Task 1.2)*
3. **Phase 2 — Core URL Shortener** *(Task 2.1 & Task 2.2)*
4. **Phase 3 — Event-Driven Analytics** *(Task 3.1 & Task 3.2)*
5. **Phase 4 — Realtime WebSocket & Moderation** *(Task 4.1 & Task 4.2)*
6. **Phase 5 — Frontend Next.js & S3/CloudFront Hosting** *(Task 5.1, Task 5.2 & Task 5.3)*
7. **Phase 6 — Production Operations, Observability & Canary CI/CD** *(Task 6.1 & Task 6.2)*

## 4. Kiểm tra nền tảng local

Yêu cầu Go 1.26.10 (pin trong `backend/go.mod` và CI). Chạy từ root:

```sh
go -C backend test -race ./...
go -C backend vet ./...
# Hoặc: make check
```

Race detector cần CGO và C compiler trên máy chạy test. CI dùng Linux; trên Windows có thể chạy qua WSL với Go/GCC. `gofmt -l backend` phải không liệt kê file nào.

`domain.GenerateShortCode(7)` dùng crypto/rand, `domain.NormalizeURL` nhận HTTP/HTTPS absolute, IP/localhost và DNS ASCII (IDN dùng punycode), không truy cập đích URL. `FormatCreatedAt` trả UTC cố định 3 chữ số milliseconds để sắp xếp GSI1.

`config.Load()` đọc `TABLE_NAME` bắt buộc; `EVENT_BUS`, `WS_ENDPOINT` và `PUBLIC_REDIRECT_BASE_URL` tùy chọn theo Lambda. Caller kiểm tra biến cần cho nghiệp vụ khi khởi tạo. `WS_ENDPOINT` là HTTP(S) Management API (ví dụ `https://abc.execute-api.ap-southeast-1.amazonaws.com/prod`), không phải địa chỉ `wss://` của browser. Base URL bỏ dấu `/` cuối, không nhận credentials/query/fragment; hỗ trợ path stage của API. Không đưa secret vào repository.

## 5. Observability và đóng gói Lambda

Mỗi invocation lấy logger theo context của Lambda để có request ID, rồi dùng cùng context cho SDK v2:

```go
logger := middleware.WithRequestID(ctx, middleware.NewLogger("resolve-link"))
logger.InfoContext(ctx, "resolved", "code", code)
if err := middleware.EmitCountMetric(os.Stdout, "UrlShortener", "resolve-link", "LinkResolved", 1); err != nil {
    logger.ErrorContext(ctx, "emit metric", "error", err)
}
```

Tạo logger gốc một lần ở startup, gọi `WithRequestID` cho từng invocation; không log token, owner ID hoặc target URL có thể chứa dữ liệu nhạy cảm. EMF xuất JSON trực tiếp ra stdout, namespace `UrlShortener`, dimension duy nhất `service`, unit Count; không đưa request ID/code vào dimension gây tăng cardinality. Hàm trả lỗi validation/write để caller có thể ghi nhận lỗi telemetry.

Gọi `middleware.InstrumentAWSConfig(&cfg)` đúng một lần trước khi tạo SDK v2 clients. Truyền context invocation vào mọi SDK call, bật Lambda Active tracing và IAM X-Ray ở Phase 2. X-Ray SDK v1.8.5 tương thích stack SDK v2 trong test local (subsegment và header propagation). SDK đã ở maintenance từ 25/02/2026, còn kéo AWS SDK v1 gián tiếp dù code ứng dụng chỉ dùng v2; transitive dependencies được pin và kiểm tra bảo mật. Scan Go1.26.10 ngày 2026-11-04 báo 0 symbol/package vulnerabilities, còn 4 module-only advisories: GO-2022-0646/GO-2022-0635 ở S3 Crypto SDK v1 và GO-2026-6866/GO-2026-6860 ở x/net; ứng dụng không import/gọi các package/symbol bị ảnh hưởng. Không suppress cảnh báo. Chưa xác minh xuất trace trên AWS. [Timeline AWS](https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-daemon-timeline.html).

Build từ root bằng một trong các lệnh:

```powershell
./scripts/build-lambdas.ps1
```

```sh
bash scripts/build-lambdas.sh
# Hoặc: make build-lambdas
```

Hai wrapper dùng chung `backend/tools/build-lambdas`, tìm các `backend/cmd/<handler>/main.go` hiện có, build `GOOS=linux GOARCH=arm64 CGO_ENABLED=0` với `lambda.norpc`, tạo `backend/dist/<handler>.zip`. Mỗi ZIP chứa đúng file gốc `bootstrap` với quyền Unix `0755`, dùng được cho `provided.al2023`/`arm64`; không cần cài zip/Python riêng. Artifact cũ được giữ nếu handler build lỗi.

Hiện có tám handler thật: create-link, resolve-link, list-links, delete-link, click-aggregator, ws-connect, ws-disconnect và moderate-check. Lệnh build tạo tám ZIP. Test dùng một chương trình Go tối thiểu trong thư mục tạm để kiểm ELF/ZIP và wrapper thực trên Windows/Linux; fixture này không phải Lambda nghiệp vụ và không chứng minh E2E AWS. Không dùng placeholder handler để tạo artifact triển khai giả.

## 6. Kiểm tra scaffold frontend tĩnh

Dùng Node26.13.0 và pnpm10.33.2 theo CI; có thể gọi đúng phiên bản pnpm qua Corepack:

```sh
corepack pnpm@10.33.2 --dir frontend install --frozen-lockfile
corepack pnpm@10.33.2 --dir frontend lint
corepack pnpm@10.33.2 --dir frontend typecheck
corepack pnpm@10.33.2 --dir frontend build
corepack pnpm@10.33.2 --dir frontend test
corepack pnpm@10.33.2 --dir frontend audit --prod
```

Build tạo `frontend/out/` để phục vụ bằng static hosting; không dùng `next start`. Chạy build trước test vì test kiểm artifact HTML/404 và asset thật. `make check` kiểm unit/race/vet backend và Terraform (yêu cầu Terraform1.17.1); `DYNAMODB_ENDPOINT=http://127.0.0.1:8000 make test-integration` kiểm DynamoDB Local và HTTP E2E; CI chạy cả backend và frontend. Frontend vẫn là scaffold: Google auth, giao diện Doodle, dashboard và API thật triển khai ở Phase 5.


## 7. Core HTTP API và kiểm thử local

Xem [hướng dẫn local](docs/local-testing.md) để chạy DynamoDB Local, HTTP harness, hai danh tính thử nghiệm và recorder EventBridge tùy chọn. Các route `POST /links`, `GET /links`, `DELETE /links/{code}` lấy owner từ JWT authorizer `sub`; `GET /{code}` công khai và chỉ trả 301 sau khi publish thành công. Không có kho dữ liệu in-memory trong luồng link.

Local harness chỉ bind loopback và dùng header `X-Local-User` nằm trong allowlist cấu hình; header này không được Lambda production dùng làm danh tính. Recorder chỉ là fixture giao thức PutEvents, không thay kiểm chứng EventBridge/IAM/Cognito hoặc pipeline analytics trên AWS. Terraform và E2E AWS vẫn pending ở Task 2.2 trở đi.


## 8. Terraform bootstrap và core infrastructure

Xem [runbook Task 2.2](docs/terraform-core-infrastructure.md) cho S3 remote state/lockfile, OIDC trust, permissions boundary, Cognito Google-only, core modules và AWS E2E Core. Chạy `make terraform-check` để build Lambda rồi fmt/init-backend=false/validate/test hai root bằng mock provider; không gọi AWS. Terraform1.17.1 và AWS provider6.71.0 được pin cùng lockfiles.

Google OAuth Web client/secret và operator authorization là đầu vào khi triển khai. AWS E2E vẫn pending; Task2.2 không chạy apply. Sau lần app deployment được phép đầu tiên, pin http_api_id trong bootstrap trước khi bật CD để thu hẹp quyền API Gateway child resources.


Hạ tầng analytics và E2E AWS: [runbook event pipeline](docs/terraform-event-pipeline.md). Bootstrap/dev có27plan tests; Scenario Pipeline mới chỉ được kiểm parser/preflight offline, chưa có evidence delivery/DLQ trên AWS.

## 8. WebSocket và moderation

Task4.1 gắn pusher khi aggregator có `WS_ENDPOINT` và cung cấp ba Lambda/ASL còn lại. Xem [hợp đồng và runbook manual](docs/websocket-and-moderation.md) cùng [bằng chứng local](docs/local-testing.md). Task4.2 đã có [Terraform và E2E scenario](docs/terraform-realtime-and-moderation.md); execution history AWS còn pending deployment được phép. Create/redirect giữ nguyên luồng độc lập với moderation.
