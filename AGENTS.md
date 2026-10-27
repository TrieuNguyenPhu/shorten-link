# AGENTS.md — Quy Chuẩn Agentic Engineering & Kiến Trúc Dự Án

## 1. Vai Trò & Mục Tiêu Cốt Lõi

Bạn đang làm việc với vai trò **Senior AWS Serverless Architect + Golang Backend Engineer + DevOps Engineer** để xây dựng hệ thống **Serverless URL Shortener with Realtime Analytics** trên AWS, tái hiện sát nhất kiến trúc, nghiệp vụ và luồng xử lý của:

- **Series 21 bài:** `https://kkloudtarus.net/blog/series/serverless-on-aws-event-driven`
- **Repository gốc:** `https://github.com/nghiadaulau/serverless-url-shortener-aws`

### Bảng Thay Đổi Bắt Buộc So Với Bản Gốc

| Thành phần | Bản gốc | Bản đích (Bắt buộc) |
| --- | --- | --- |
| Lambda Runtime | Node.js 22 / TypeScript (`esbuild`) | **Golang** (`provided.al2023`, `arm64`, binary `bootstrap`) |
| Infrastructure as Code | AWS SAM (`template.yaml`) | **Terraform (HCL)** quản lý 100% tài nguyên |
| Frontend Dashboard | `web/index.html` (Vanilla HTML/JS) | **Next.js App Router + TypeScript + Tailwind CSS** (`output: "export"`) |
| Frontend Hosting | Mở file tĩnh cục bộ | **Amazon S3 (Private) + CloudFront với Origin Access Control (OAC)** |
| Các thành phần AWS khác | DynamoDB, Cognito, HTTP API, WebSocket API, EventBridge, SQS + DLQ, Step Functions, CloudWatch/X-Ray, SNS, CodeDeploy Canary | **Giữ dịch vụ cốt lõi, tên bảng/event schema và access patterns; các sửa lỗi có chủ đích được chốt trong đặc tả kiến trúc** |

### Các Ràng Buộc Nghiêm Cấm Vi Phạm

1. **Không thay đổi dịch vụ cốt lõi:** Không thay DynamoDB bằng PostgreSQL/MongoDB; không thay EventBridge bằng Kafka; không chuyển Lambda sang ECS/EKS; không thêm Redis hoặc microservice framework dư thừa.
2. **Static Frontend thuần túy:** Next.js bắt buộc cấu hình `output: "export"`. Tuyệt đối không dùng SSR, Server Actions, Next.js API Routes hoặc bất kỳ tính năng nào cần Node.js runtime khi chạy trên S3 + CloudFront.
3. **Mỗi nghiệp vụ một Lambda riêng:** Biên dịch từng handler trong `backend/cmd/<handler-name>` thành file thực thi Linux `arm64` tên `bootstrap` đóng gói ZIP độc lập:
   - `create-link` (`POST /links`)
   - `resolve-link` (`GET /{code}`)
   - `list-links` (`GET /links`)
   - `delete-link` (`DELETE /links/{code}`)
   - `click-aggregator` (SQS `ClickQueue` consumer)
   - `ws-connect` (WebSocket `$connect`)
   - `ws-disconnect` (WebSocket `$disconnect`)
   - `moderate-check` (Step Functions task `CheckSafety`)
4. **Quy tắc ngôn ngữ:**
   - Tài liệu, giải thích, báo cáo tiến độ, nội dung PR viết bằng **tiếng Việt**.
   - Tên biến, hàm, package, file, commit message và mã nguồn viết bằng **tiếng Anh** chuẩn.
5. **Không dùng mock data cho luồng chính:** Mọi luồng nghiệp vụ phải đi qua code thực thi thật, có unit test, integration test và kịch bản E2E kiểm chứng.
6. **An toàn chi phí AWS:** Không tự ý chạy `terraform apply` hoặc chạy load test lớn trên AWS khi chưa được người dùng yêu cầu rõ ràng.

---

## 2. Hệ Sinh Thái 13 Bộ Skills Đã Cài Đặt (`~/.agents/skills` & `.agents/skills.json`)

Kiểm tra ngày 2026-10-26 ghi nhận 314 thư mục cấp đầu có `SKILL.md`, 376 file `SKILL.md` khi đếm đệ quy (có bản trùng), cùng file đại diện cho 13 bộ dưới đây tại `~/.agents/skills`. Hai cấu hình `.agents/skills.json` và `~/.gemini/config/skills.json` đều trỏ tới thư mục này. Đây là bằng chứng hiện diện/cấu hình, không bảo đảm mọi host tự nạp skill hay xác thực độc lập nguồn gốc và phiên bản. Agent đọc và kích hoạt skill phù hợp với task; xem [bằng chứng audit](docs/phase-0-reference-audit.md#6-bằng-chứng-skills-và-giới-hạn-xác-minh).

| # | Nguồn Skill / Framework | Các Skill Tiêu Biểu | Khi Nào Sử Dụng |
| --- | --- | --- | --- |
| 1 | `obra/superpowers` | `writing-plans`, `test-driven-development`, `subagent-driven-development`, `executing-plans`, `verification-before-completion`, `requesting-code-review`, `finishing-a-development-branch` | Lập kế hoạch chi tiết, thực thi TDD (Red-Green-Refactor), xác minh trước khi đóng task và hoàn tất branch. |
| 2 | `DietrichGebert/ponytail` | `ponytail`, `ponytail-review`, `ponytail-audit`, `ponytail-debt` | Giữ code gọn, đúng trọng tâm (YAGNI), không viết wrapper/abstraction thừa nhưng không bao giờ cắt bỏ validation, xử lý lỗi, bảo mật và test. |
| 3 | `OthmanAdi/planning-with-files` | `planning-with-files` | Duy trì bộ nhớ làm việc bền vững trên đĩa qua `task_plan.md`, `findings.md`, và `progress.md`. |
| 4 | `JuliusBrussee/caveman` | `investigate-first`, `lean-build`, `surgical-patch`, `safe-refactor`, `verify-and-stop`, `caveman-review` | Điều tra trước khi sửa, thay đổi tối thiểu chính xác, kiểm tra bằng chứng thực thi. |
| 5 | `trailofbits/skills` | `spec-to-code-compliance`, `differential-review`, `sharp-edges`, `supply-chain-risk-auditor`, `gh-cli`, `property-based-testing` | Kiểm tra độ khớp giữa đặc tả và code (DynamoDB conditions, IDOR, IAM least-privilege), review bảo mật diff trước khi mở PR. |
| 6 | `vercel-labs/agent-skills` | `react-best-practices`, `composition-patterns`, `web-design-guidelines` | Tối ưu kiến trúc component React/Next.js App Router, hook WebSocket và hiệu năng render. |
| 7 | `anthropics/skills` | `frontend-design`, `webapp-testing`, `doc-coauthoring` | Thiết kế giao diện chất lượng cao, kiểm thử ứng dụng web và biên soạn tài liệu kỹ thuật. |
| 8 | `gsd-build/get-shit-done` (`open-gsd/gsd-core`) | `gsd-plan-phase`, `gsd-execute-phase`, `gsd-verify-work`, `gsd-code-review`, `gsd-pr-branch`, `gsd-progress` | Điều phối tiến độ theo Phase, kiểm tra tiêu chí hoàn thành (Definition of Done) của từng giai đoạn. |
| 9 | `Egonex-AI/Understand-Anything` | `understand`, `understand-diff`, `understand-domain`, `understand-explain` | Phân tích cấu trúc phụ thuộc giữa các package Go, module Terraform và luồng sự kiện. |
| 10 | `alibaba/open-code-review` | `open-code-review`, `open-code-review-delegate` | Thực hiện rà soát chất lượng code và độ chính xác theo dòng trước khi tạo Pull Request. |
| 11 | `nextlevelbuilder/ui-ux-pro-max-skill` | `ui-ux-pro-max`, `design-system`, `ui-styling` | Xây dựng hệ thống màu sắc, typography, trạng thái loading/error/realtime cho Dashboard Next.js. |
| 12 | `Leonxlnx/taste-skill` | `taste-skill` | Đảm bảo thẩm mỹ UI hiện đại, sạch sẽ, tránh giao diện AI chung chung. |
| 13 | `shanraisshan/claude-code-best-practice` | Quy trình `Research → Plan → Execute → Review → Ship` | Tổ chức điều phối công việc có kỷ luật, kiểm chứng từng bước trước khi chuyển giai đoạn. |

---

## 3. Quy Trình Thực Thi Task, Git Branch, Pull Request & Cập Nhật Trạng Thái

Mỗi Task trong kế hoạch (`docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md` và `task_plan.md`) **BẮT BUỘC** tuân theo vòng đời 6 bước sau:

### Bước 1: Đồng bộ trạng thái (`planning-with-files`)
- Đọc `PROJECT_STATUS.md`, `task_plan.md`, `findings.md`, và `progress.md` trước khi bắt đầu task.
- Đánh dấu task hiện tại là `in_progress` trong `task_plan.md` và `PROJECT_STATUS.md`.

### Bước 2: Tạo nhánh riêng cho Task
- Luôn tạo branch mới từ `main` theo quy ước:
  - `chore/phase-00-task-01-...`
  - `feat/phase-01-task-02-...`
  - `feat/phase-02-task-03-...`
- Tuyệt đối không gộp nhiều task độc lập vào chung một PR.
- Sửa review của cùng task tiếp tục trên branch/PR đang mở; không tạo branch hay PR mới chỉ để đóng review.

### Bước 3: Phát triển theo TDD (`test-driven-development` + `ponytail`)
- Viết unit test / integration test trước để định nghĩa hành vi.
- Triển khai code tối giản, rõ ràng, đáp ứng đầy đủ đặc tả kỹ thuật.
- Chạy bộ kiểm tra tương ứng:
  - Backend Go: `go test -race ./...` và `go vet ./...`
  - Frontend Next.js: `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build` (kiểm tra xuất tĩnh `out/`)
  - Infrastructure Terraform: `terraform fmt -check -recursive`, `terraform init -backend=false`, `terraform validate` và các `terraform test` đã định nghĩa
- Task chỉ sửa tài liệu kiểm tra diff, liên kết và ví dụ; không tuyên bố đã chạy bộ kiểm thử ứng dụng khi chưa có implementation. Unit test dùng mock ở ranh giới phụ thuộc; integration dùng DynamoDB Local/container hoặc AWS đã được phép.

### Bước 4: Rà soát bảo mật & chất lượng (`differential-review` + `ponytail-review`)
- Kiểm tra `git diff` để đảm bảo:
  - Không lộ secret, `.tfstate`, `.tfvars` thực tế hoặc file rác.
  - Đúng chuẩn IAM least-privilege và chống IDOR bằng `ConditionExpression` trong DynamoDB.
  - Xử lý đúng `TransactionCanceledException.CancellationReasons` trong `click-aggregator`.

### Bước 5: Cập nhật trạng thái dự án (Bắt buộc trước khi commit cuối của Task)
- Cập nhật `PROJECT_STATUS.md` (trạng thái Phase/Task hiện tại, những gì đã hoàn thành, lệnh kiểm tra đã chạy, bước tiếp theo).
- Cập nhật `task_plan.md` (đánh dấu `[x]` và `Status: complete` cho task/phase tương ứng).
- Ghi nhật ký kết quả test và bài học kỹ thuật vào `progress.md` và `findings.md`.

### Bước 6: Tạo Pull Request riêng cho Task
- Push branch lên `origin` và tạo Pull Request riêng bằng `gh pr create` với mô tả tiếng Việt rõ ràng (Mục tiêu task, Thay đổi chính, Kết quả kiểm thử, Trạng thái dự án).
- Ghi nhận URL/số PR vào `PROJECT_STATUS.md` và `progress.md`.

---

## 4. Cấu Trúc Thư Mục Chuẩn Của Monorepo

```text
shorten-link/
├── frontend/                  # Next.js App Router (output: "export") + TypeScript + Tailwind CSS
│   ├── src/
│   │   ├── app/
│   │   ├── components/
│   │   ├── features/
│   │   ├── hooks/
│   │   ├── lib/
│   │   └── types/
│   └── package.json
├── backend/                   # Golang (AWS SDK v2, aws-lambda-go, provided.al2023 arm64)
│   ├── cmd/
│   │   ├── create-link/
│   │   ├── resolve-link/
│   │   ├── list-links/
│   │   ├── delete-link/
│   │   ├── click-aggregator/
│   │   ├── ws-connect/
│   │   ├── ws-disconnect/
│   │   └── moderate-check/
│   ├── internal/
│   │   ├── domain/
│   │   ├── service/
│   │   ├── repository/
│   │   ├── events/
│   │   ├── middleware/
│   │   └── config/
│   ├── go.mod
│   └── go.sum
├── infrastructure/
│   └── terraform/
│       ├── bootstrap/         # S3 remote state + versioning + lockfile
│       ├── modules/           # Các module Terraform theo chức năng
│       └── environments/
│           └── dev/           # Cấu hình môi trường triển khai (mặc định ap-southeast-1)
├── statemachine/
│   └── moderation.asl.json    # Định nghĩa AWS Step Functions Standard
├── scripts/                   # Script build Lambda arm64 ZIP, deploy helper, local harness
├── tests/
│   └── load/                  # Kịch bản k6 load test cho GET /{code}
├── docs/                      # Tài liệu audit bản gốc, kiến trúc, hợp đồng API và kế hoạch
├── .agents/
│   └── skills.json            # Kết nối kho 13 bộ skills
├── .github/
│   └── workflows/             # CI/CD GitHub Actions (OIDC, Canary CodeDeploy, S3 + CloudFront)
├── AGENTS.md                  # File quy chuẩn này
├── PROJECT_STATUS.md          # Bảng theo dõi trạng thái dự án thời gian thực
├── task_plan.md               # Kế hoạch thực thi theo planning-with-files
├── findings.md                # Ghi chú phân tích kỹ thuật & quyết định thiết kế
├── progress.md                # Nhật ký tiến độ từng phiên và từng PR
├── Makefile
└── README.md
```
