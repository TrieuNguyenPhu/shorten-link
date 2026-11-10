# Shrnk — Serverless URL Shortener & Realtime Analytics

Hệ thống rút gọn URL chuẩn cloud-native serverless, hiệu năng cao với độ trễ chuyển hướng (redirect) cực thấp, kiến trúc hướng sự kiện (event-driven architecture) để thu thập analytics bất đồng bộ và đẩy dữ liệu thống kê thời gian thực (realtime) tới người dùng thông qua WebSocket.

Dự án được xây dựng toàn diện trên **AWS** với **Go Lambda (arm64)**, **DynamoDB**, **EventBridge**, **SQS/DLQ**, **API Gateway HTTP/WebSocket**, **Step Functions**, **Cognito (Google OAuth PKCE)**, **CloudFront OAC + S3**, và quản lý hạ tầng bằng **Terraform** chuẩn hóa với Permission Boundaries.

🌐 **Production Website:** [https://npt-shortenlink.dev](https://npt-shortenlink.dev) *(Fallback: [CloudFront Distribution](https://d2aamyg2mt28wl.cloudfront.net))*

---

## 📑 Mục lục

1. [Kiến trúc hệ thống (Architecture)](#-kiến-trúc-hệ-thống-architecture)
2. [Tính năng nổi bật (Key Features)](#-tính-năng-nổi-bật-key-features)
3. [Công nghệ sử dụng (Tech Stack)](#-công-nghệ-sử-dụng-tech-stack)
4. [Cấu trúc mã nguồn (Project Structure)](#-cấu-trúc-mã-nguồn-project-structure)
5. [Cơ chế bảo mật & Tuân thủ (Security & Compliance)](#-cơ-chế-bảo-mật--tuân-thủ-security--compliance)
6. [Hướng dẫn cài đặt & Chạy cục bộ (Local Development)](#-hướng-dẫn-cài-đặt--chạy-cục-bộ-local-development)
7. [Kiểm thử (Testing & Quality Gates)](#-kiểm-thử-testing--quality-gates)
8. [Triển khai hạ tầng (Terraform Deployment)](#-triển-khai-hạ-tầng-terraform-deployment)
9. [CI/CD & Chiến lược phát hành Canary (Canary Deployment)](#-cicd--chiến-lược-phát-hành-canary)
10. [API & WebSocket Contracts](#-api--websocket-contracts)

---

## 🏗 Kiến trúc hệ thống (Architecture)

```mermaid
flowchart TD
    subgraph Client ["Client Layer"]
        User["Người dùng / Trình duyệt"]
    end

    subgraph Edge ["Edge & CDN Layer"]
        CF["AWS CloudFront (OAC)"]
        CF_FN["CloudFront Functions (URL Rewrite)"]
        S3["AWS S3 Bucket (Private Frontend Export)"]
        CustomDomain["Custom Domain (npt-shortenlink.dev)"]
        CustomDomain --> CF
        CF --> CF_FN --> S3
    end

    subgraph Auth ["Authentication"]
        Cognito["Amazon Cognito User Pool"]
        GoogleIDP["Google Identity Provider (OAuth 2.0 PKCE)"]
        GoogleIDP --> Cognito
    end

    subgraph APILayer ["API Gateway Layer"]
        HttpAPI["API Gateway (HTTP API v2)"]
        WsAPI["API Gateway (WebSocket API)"]
    end

    subgraph CoreBackend ["Backend Serverless (Go arm64)"]
        FnCreate["Lambda: create-link"]
        FnResolve["Lambda: resolve-link"]
        FnList["Lambda: list-links"]
        FnDelete["Lambda: delete-link"]
        FnWsConnect["Lambda: ws-connect"]
        FnWsDisconnect["Lambda: ws-disconnect"]
        FnAggregator["Lambda: click-aggregator"]
        FnModerate["Lambda: moderate-check"]
    end

    subgraph AsyncPipeline ["Event & Analytics Pipeline"]
        EventBus["AWS EventBridge (shortener.events)"]
        ClickQueue["Amazon SQS (ClickQueue)"]
        ClickDLQ["Amazon SQS (ClickDLQ)"]
        EventBus -->|LinkRedirected Event| ClickQueue
        ClickQueue -.->|Max Receive Count Exceeded| ClickDLQ
        ClickQueue -->|Batch Event Source| FnAggregator
    end

    subgraph Moderation ["Content Moderation Engine"]
        StepFn["AWS Step Functions (moderation.asl.json)"]
        StepFn --> FnModerate
    end

    subgraph Database ["Data Storage"]
        DDB[("Amazon DynamoDB (Single Table Design)")]
    end

    %% Client Interactions
    User -->|Xem giao diện web| CF
    User -->|Đăng nhập Google PKCE| Cognito
    User -->|Tạo / Xem / Xóa link| HttpAPI
    User -->|Truy cập link rút gọn| HttpAPI
    User -->|Mở kết nối Realtime| WsAPI

    %% HTTP API Routing
    HttpAPI -->|POST /links| FnCreate
    HttpAPI -->|GET /{code}| FnResolve
    HttpAPI -->|GET /links| FnList
    HttpAPI -->|DELETE /links/{code}| FnDelete

    %% Backend DynamoDB Operations
    FnCreate -->|PutItem LINK#code| DDB
    FnResolve -->|GetItem LINK#code & 302 Redirect| DDB
    FnList -->|Query GSI1 OwnerIndex| DDB
    FnDelete -->|DeleteItem| DDB

    %% Realtime WebSocket Routing
    WsAPI -->|$connect| FnWsConnect
    WsAPI -->|$disconnect| FnWsDisconnect
    FnWsConnect -->|Save Connection| DDB
    FnWsDisconnect -->|Remove Connection| DDB

    %% Async Telemetry & Push
    FnResolve -.->|Asynchronous PutEvents| EventBus
    FnAggregator -->|TransactWriteItems: Inc Clicks + Save Log| DDB
    FnAggregator -->|Push Realtime Metrics PostToConnection| WsAPI
    WsAPI -.->|Push notification| User

    %% Moderation Flow
    FnCreate -.->|Trigger Moderation| StepFn
    StepFn -->|Update Status: active / rejected| DDB
```

---

## ⚡ Tính năng nổi bật (Key Features)

- **Độ trễ chuyển hướng cực thấp (Low Latency 302 Redirects):**
  Lambda `resolve-link` đọc bản ghi từ DynamoDB và trả về ngay mã phản hồi `302 Found` (best-effort redirect). Quá trình ghi nhận click được phát hành bất đồng bộ vào EventBridge, hoàn toàn không chặn hay gây trễ cho người truy cập.
- **Xử lý số liệu bất đồng bộ & Giao dịch 2-item an toàn (Event-Driven Analytics):**
  Sự kiện `LinkRedirected` được truyền qua EventBridge tới SQS. Lambda `click-aggregator` nhận batch sự kiện và thực thi giao dịch nguyên tử DynamoDB (`TransactWriteItems`):
  1. Tăng số đếm click tổng hợp (`ADD clicks :inc`).
  2. Ghi nhận chi tiết lượt click (`CLICK#<timestamp>`).
- **Dashboard thời gian thực (Realtime Live Clicks):**
  Frontend kết nối WebSocket qua API Gateway. Mỗi khi link được truy cập, `click-aggregator` tự động tra cứu các kết nối active của chủ sở hữu link và phát trực tiếp (broadcast) số click mới nhất tới trình duyệt qua API Gateway Management API.
  - Tích hợp cơ chế kết nối lại có giới hạn (bounded reconnect) với exponential backoff và jitter để tránh DDoS/thundering herd.
- **Kiểm duyệt nội dung tự động (Automated Content Moderation):**
  AWS Step Functions điều phối quy trình kiểm tra URL đích thông qua Lambda `moderate-check`. Hệ thống lọc từ khóa độc hại (`malware`, `phishing`, `evil`) với cơ chế phân tách token từ vựng chính xác (tránh false positive với các từ như `medieval` hay `devil`), sau đó tự động cập nhật trạng thái link (`active` hoặc `rejected`).
- **Chống tấn công SSRF & Chuẩn hóa URL nghiêm ngặt:**
  Xác thực URL đích theo RFC, giới hạn độ dài tối đa 2048 ký tự, chặn toàn bộ dải IP riêng tư (RFC 1918), loopback, link-local, multicast, IP metadata của cloud provider (`169.254.169.254`) và các domain nội bộ (`.local`, `.localhost`).
- **Xác thực Google OAuth 2.0 PKCE & Tuân thủ PDPL:**
  Người dùng xác thực trực tiếp với Amazon Cognito thông qua Google IDP bằng luồng PKCE an toàn, lưu trữ phiên trong `sessionStorage`, kiểm tra đối soát `state` và `nonce`.
  Tích hợp màn hình/modal xin chấp thuận quyền riêng tư (PDPL/GDPR Consent) trước khi chuyển hướng sang màn hình đăng nhập Google.
- **Phát hành an toàn với Canary Rollout:**
  Lambda `resolve-link` được quản lý bởi AWS CodeDeploy Canary deployment, theo dõi CloudWatch Alarms để tự động rollback khi phát hiện tỷ lệ lỗi gia tăng.

---

## 🛠 Công nghệ sử dụng (Tech Stack)

| Thành phần | Công nghệ / Dịch vụ | Ghi chú |
| :--- | :--- | :--- |
| **Backend Core** | Go 1.26.x, AWS Lambda Go SDK v2 | Kiến trúc arm64 (`provided.al2023`), binary tối ưu (`-ldflags="-s -w"`) |
| **Database** | Amazon DynamoDB | Single-table design, On-Demand Capacity, GSI cho phân trang user links |
| **Event Pipeline** | Amazon EventBridge, Amazon SQS + DLQ | Xử lý sự kiện bất đồng bộ, dead-letter queue lưu vết sự kiện lỗi |
| **Realtime** | Amazon API Gateway WebSocket API | Quản lý kết nối hai chiều thời gian thực |
| **Workflow** | AWS Step Functions | State machine kiểm duyệt link bất đồng bộ |
| **Auth** | Amazon Cognito User Pool + Google IDP | Chuẩn OAuth 2.0 PKCE, JWT Authorizer trên HTTP API |
| **Frontend** | Next.js 16 (App Router), React 19, TypeScript | Xuất tĩnh (`output: 'export'`), Tailwind CSS v4 |
| **Hosting & CDN** | AWS S3 + AWS CloudFront (OAC) | Bucket private, CloudFront Function rewrite URL, Custom Domain + ACM |
| **IaC** | Terraform 1.17.1 | Cấu trúc module hóa, tích hợp IAM Permission Boundaries |
| **Observability** | AWS CloudWatch, AWS X-Ray, SNS | Lưu trữ log 365 ngày, distributed tracing, metric alarms & email alerts |
| **CI/CD** | GitHub Actions | Kiểm thử đa tầng (Go race/vet, DynamoDB Local, Node tests, Terraform test) |

---

## 📂 Cấu trúc mã nguồn (Project Structure)

```text
shorten-link/
├── .github/
│   └── workflows/
│       ├── ci.yml                 # CI: lint, unit test, integration test, tf test
│       └── deploy.yml             # CD: kế hoạch phân tầng plan/apply an toàn cho dev
├── backend/
│   ├── cmd/                       # 8 Lambda Handlers (Go)
│   │   ├── create-link/           # Tạo link rút gọn mới & kiểm tra SSRF
│   │   ├── resolve-link/          # 302 Redirect & gửi event bất đồng bộ
│   │   ├── list-links/            # Phân trang cursor-based danh sách link
│   │   ├── delete-link/           # Xóa link thuộc quyền sở hữu
│   │   ├── click-aggregator/      # Nhận SQS batch, ghi transaction DDB & push WS
│   │   ├── ws-connect/            # Lưu trữ connectionId của WebSocket
│   │   ├── ws-disconnect/         # Xóa connectionId khi ngắt kết nối
│   │   └── moderate-check/        # Kiểm tra nội dung URL theo Step Functions
│   ├── internal/
│   │   ├── domain/                # Model, tạo shortcode Base62, SSRF validators
│   │   ├── repository/            # DynamoDB CRUD, TransactWrite, GSI queries
│   │   ├── service/               # Nghiệp vụ core, WebSocket broadcasting
│   │   ├── events/                # EventBridge publisher & WebSocket pusher
│   │   ├── httpapi/               # Bộ điều phối request HTTP API v2 & CORS
│   │   ├── wsapi/                 # Điều phối kết nối WebSocket
│   │   └── middleware/            # Logger chuẩn hóa, AWS X-Ray tracing, EMF metrics
│   ├── tools/
│   │   ├── build-lambdas/         # Tool build toàn bộ Lambda arm64 ra backend/dist/
│   │   └── local-http/            # Mock server chạy kiểm thử tích hợp không cần AWS
│   └── dist/                      # Nơi chứa các file .zip triển khai của Lambda
├── frontend/                      # Ứng dụng Next.js (Static Export)
│   ├── src/
│   │   ├── app/                   # App Router: Home, Login, Privacy, Dashboard, Callback
│   │   ├── components/            # UI components (Navbar, Footer, GoogleLoginButton, ...)
│   │   ├── features/links/        # Bảng danh sách link, form tạo link, workspace
│   │   ├── hooks/                 # useRealtimeClicks (quản lý WebSocket & state)
│   │   └── lib/                   # API client, PKCE Auth helpers, public runtime config
│   └── tests/                     # Kiểm thử Jest/Node native cho client, auth, UI
├── infrastructure/
│   └── terraform/
│       ├── bootstrap/             # S3 backend, DynamoDB locks, IAM Permission Boundaries
│       ├── environments/dev/      # Môi trường dev kết nối toàn bộ modules
│       └── modules/               # Modules tái sử dụng: dynamodb, lambda, http_api,
│                                  # websocket_api, event_pipeline, cognito,
│                                  # frontend_hosting, moderation_sfn, observability, ...
├── scripts/
│   ├── deploy-app.sh              # Kịch bản triển khai Terraform với plan lưu trữ S3
│   ├── deploy-canary.sh           # Điều phối CodeDeploy canary cho resolve-link
│   ├── publish-frontend.sh        # Build tĩnh, đồng bộ S3 & invalidate CloudFront
│   └── tests/                     # Unit test bằng Python cho các scripts vận hành
├── statemachine/
│   └── moderation.asl.json        # Định nghĩa Amazon States Language cho Step Functions
├── tests/
│   ├── e2e/                       # Kịch bản kiểm thử E2E hosting và canary
│   └── load/                      # Kịch bản kiểm thử tải k6 cho luồng redirect
└── Makefile                       # Tự động hóa các tác vụ test, vet, check, build
```

---

## 🔒 Cơ chế bảo mật & Tuân thủ (Security & Compliance)

1. **SSRF Guard (Server-Side Request Forgery):**
   Mọi liên kết trước khi được chấp thuận đều đi qua hàm `domain.ValidateTargetURL`:
   - Phải bắt đầu bằng giao thức `http://` hoặc `https://`.
   - Ngăn chặn tuyệt đối các host nội bộ: `localhost`, `*.localhost`, `*.local`.
   - Ngăn chặn các địa chỉ IP Loopback (`127.0.0.0/8`, `::1`), Private Network (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`), Link-local unicast/multicast và IP AWS Metadata (`169.254.169.254`).
2. **IAM Least Privilege & Permission Boundaries:**
   Tất cả IAM Roles dành cho Lambda đều bị ràng buộc bởi IAM Permission Boundaries (`runtime-boundary`, `moderation-boundary`, `canary-boundary`) khởi tạo từ tầng `bootstrap`. Các hàm chỉ có quyền trên đúng tài nguyên ARN cụ thể của bảng DynamoDB hoặc Event Bus được chỉ định.
3. **Private Frontend Delivery (CloudFront OAC):**
   S3 Bucket chứa frontend tĩnh không mở public. CloudFront sử dụng Origin Access Control (OAC) với bucket policy nghiêm ngặt chỉ cấp quyền đọc duy nhất cho CloudFront distribution ARN.
4. **Bảo mật xác thực & PDPL Compliance:**
   - Sử dụng Google OAuth 2.0 PKCE: mã hóa sinh code verifier & code challenge ngay tại trình duyệt, lưu trong `sessionStorage`, xóa sạch ngay khi kết thúc vòng đời callback.
   - Tuân thủ bảo vệ dữ liệu cá nhân (PDPL): yêu cầu người dùng tích chọn đồng ý điều khoản xử lý dữ liệu trước khi kích hoạt chuyển hướng sang Google login.
5. **Observability & Nhật ký lưu trữ:**
   - CloudWatch Logs được cấu hình retention 365 ngày cho toàn bộ Lambda và API Gateway access logs.
   - Tích hợp AWS X-Ray tracing giúp truy vết request phân tán qua từng khâu (API Gateway -> Lambda -> DynamoDB / EventBridge).

---

## 💻 Hướng dẫn cài đặt & Chạy cục bộ (Local Development)

### Yêu cầu tiên quyết (Prerequisites)
- **Go:** `1.26.x`
- **Node.js:** `>= 20.x` (khuyến nghị `26.x`)
- **pnpm:** `10.33.2` (sử dụng qua Corepack)
- **Terraform:** `1.17.1`
- **Docker:** (Nếu muốn chạy DynamoDB Local)

### 1. Khởi chạy Frontend cục bộ
```bash
cd frontend
corepack pnpm@10.33.2 install --frozen-lockfile
corepack pnpm@10.33.2 dev
```
Giao diện frontend sẽ chạy tại `http://localhost:3000`.

### 2. Build toàn bộ Lambda Handlers
Hệ thống sử dụng công cụ Go tự viết để biên dịch song song tất cả 8 Lambda handlers sang kiến trúc ARM64 và nén thành file zip:
```bash
make build-lambdas
# Hoặc: go -C backend run ./tools/build-lambdas
```
Kết quả được tạo ra trong thư mục `backend/dist/*.zip`.

---

## 🧪 Kiểm thử (Testing & Quality Gates)

Dự án áp dụng hệ thống kiểm tra toàn diện trước khi phát hành:

```bash
# Chạy toàn bộ kiểm tra chất lượng (Go race test, vet, terraform check, frontend check, operations check)
make check
```

Từng lệnh kiểm tra riêng lẻ:

### Backend Unit & Race Tests
```bash
make test
# Hoặc: go -C backend test -race ./...
go -C backend vet ./...
```

### Backend Integration Tests với DynamoDB Local
Khởi động container DynamoDB Local:
```bash
docker run -d -p 8000:8000 amazon/dynamodb-local:3.3.0
```
Chạy kiểm thử tích hợp (CRUD, Transactions, WebSocket Dispatch):
```bash
DYNAMODB_ENDPOINT=http://127.0.0.1:8000 make test-integration
```

### Frontend Tests & Typecheck
```bash
corepack pnpm@10.33.2 --dir frontend lint
corepack pnpm@10.33.2 --dir frontend typecheck
corepack pnpm@10.33.2 --dir frontend test
corepack pnpm@10.33.2 --dir frontend build
```

### Terraform Validation & Unit Tests
```bash
make terraform-check
```

---

## 🚀 Triển khai hạ tầng (Terraform Deployment)

### 1. Khởi tạo tầng Bootstrap (Một lần duy nhất)
Tầng `bootstrap` tạo S3 bucket lưu remote state, DynamoDB lock table và các IAM Permission Boundaries:
```bash
cd infrastructure/terraform/bootstrap
terraform init
terraform apply
```

### 2. Triển khai môi trường Dev
Tạo file `infrastructure/terraform/environments/dev/terraform.tfvars` dựa trên file mẫu `terraform.tfvars.example`:
```hcl
aws_account_id           = "123456789012"
aws_region               = "ap-southeast-1"
name_prefix              = "url-shortener-dev"
cognito_domain_prefix    = "shrnk-auth-dev"
google_client_id         = "your-google-client-id"
google_client_secret     = "your-google-client-secret"
frontend_domain_name     = "npt-shortenlink.dev"
frontend_certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/xxx"
alarm_email_addresses    = ["admin@example.com"]
```

> [!NOTE]
> Chứng chỉ ACM (SSL Certificate) cho CloudFront bắt buộc phải được tạo tại region `us-east-1` (N. Virginia).

Thực hiện triển khai:
```bash
cd infrastructure/terraform/environments/dev
terraform init -backend-config="bucket=<YOUR_TF_STATE_BUCKET>" -backend-config="region=ap-southeast-1"
terraform plan -out=tfplan
terraform apply tfplan
```

### 3. Xuất bản Frontend lên S3 & CloudFront
Sử dụng script tự động hóa để đồng bộ các asset và làm mới cache CloudFront:
```bash
bash scripts/publish-frontend.sh
```

---

## 🔄 CI/CD & Chiến lược phát hành Canary

- **Continuous Integration (`.github/workflows/ci.yml`):**
  Tự động kích hoạt khi có Pull Request hoặc push vào nhánh `main`. Quy trình kiểm tra bao gồm:
  - Khởi tạo service `dynamodb-local` để chạy toàn bộ integration tests.
  - Kiểm tra formatting, race detector, Go vet, và `govulncheck`.
  - Kiểm tra Next.js lint, typecheck, static build export và node test suite.
  - Kiểm tra cú pháp Terraform, validate và mock tests (`terraform test`).
  - Kiểm tra tính hợp lệ của kịch bản shell deployment bằng Python tests và Actionlint.

- **Continuous Deployment (`.github/workflows/deploy.yml`):**
  Sử dụng quy trình 2 bước với phân quyền môi trường nghiêm ngặt:
  1. **Tạo Plan (`action: plan`):** Tạo ra file `.tfplan` được ký SHA256 và lưu trực tiếp vào S3 private bucket (tuyệt đối không lộ ra GitHub artifact).
  2. **Áp dụng Plan (`action: apply`):** Chỉ người có thẩm quyền sau khi review plan mới có thể kích hoạt apply bằng `plan_run_id` tương ứng tại đúng commit của nhánh `main`.
  3. **Canary Verification (`scripts/deploy-canary.sh`):** CodeDeploy thực hiện định tuyến lưu lượng dần sang phiên bản Lambda `resolve-link` mới, tự động hủy bỏ và hoàn tác nếu CloudWatch alarm kích hoạt.

---

## 📡 API & WebSocket Contracts

### HTTP API Endpoints (Base URL: `https://<api-id>.execute-api.ap-southeast-1.amazonaws.com`)

| Phương thức | Đường dẫn | Xác thực | Mô tả |
| :--- | :--- | :--- | :--- |
| `POST` | `/links` | Bearer Token (Cognito) | Tạo link rút gọn. Body: `{"url": "https://...", "alias": "my-link", "expiryDays": 30}`; `alias` và `expiryDays` tùy chọn, bỏ trống thời hạn là vĩnh viễn. |
| `GET` | `/{code}` | Không | Chuyển hướng 302 tới URL gốc và phát hành sự kiện analytics |
| `GET` | `/links` | Bearer Token (Cognito) | Liệt kê danh sách link của tài khoản (hỗ trợ query param `?cursor=...`) |
| `DELETE` | `/links/{code}` | Bearer Token (Cognito) | Xóa link thuộc quyền sở hữu của người dùng |

#### Ví dụ tạo link (`POST /links`):
**Request:**
```http
POST /links HTTP/1.1
Authorization: Bearer <cognito_id_token>
Content-Type: application/json

{
  "url": "https://example.com/very-long-url-path",
  "alias": "docs-example",
  "expiryDays": 30
}
```

**Response (201 Created):**
```json
{
  "code": "docs-example",
  "target": "https://example.com/very-long-url-path",
  "shortUrl": "https://<api-id>.execute-api.ap-southeast-1.amazonaws.com/docs-example",
  "status": "pending",
  "expiresAt": 1794000000
}
```

`alias` phải dài 3–32 ký tự chữ, số hoặc dấu gạch ngang, không bắt đầu/kết thúc bằng dấu gạch ngang và không dùng tên route `links`. `expiryDays` nhận số nguyên từ 1 đến 365; khi không gửi trường này, link không hết hạn. Link hết hạn trả về `404` ngay cả khi DynamoDB chưa dọn bản ghi TTL.

---

### WebSocket API (`wss://<ws-id>.execute-api.ap-southeast-1.amazonaws.com/prod`)

- **Kết nối `$connect`:**
  Client gửi query string chứa token: `wss://...?token=<cognito_token>`
  Server lưu trữ `connectionId` và ánh xạ tới `ownerId` trong DynamoDB.
- **Push Payload từ Server (`click-aggregator`):**
  Khi phát sinh lượt click hợp lệ, server đẩy thông điệp realtime tới các client đang lắng nghe:
  ```json
  {
    "type": "click_update",
    "code": "aB3x9Z1",
    "clicks": 42,
    "timestamp": "2026-11-05T14:30:00.000Z"
  }
  ```
- **Ngắt kết nối `$disconnect`:**
  Tự động dọn dẹp bản ghi connection khỏi DynamoDB để giải phóng tài nguyên.

---

## 📄 Bản quyền & Tác giả (License & Credits)

- Được phát triển bởi **Trieu Nguyen** ([@TrieuNguyenPhu](https://github.com/TrieuNguyenPhu)).
- Tham khảo kiến trúc Serverless Event-Driven từ series hướng dẫn AWS Serverless.
