# Audit nợ kỹ thuật và nợ hạ tầng — 2026-11-10

Phạm vi: backend Go, frontend Next.js, Terraform bootstrap/dev, CI/CD, deployment scripts và môi trường AWS dev.

## Trạng thái sau remediation

- GitHub: PR #95, #96, #98 và #99 đã merge vào `main`; commit kiểm tra cuối là `ac09bff1ecc45bca2e6c749dbe2b459d55dcc49f`.
- AWS: CodeDeploy deployment `d-1U4K1T38M` succeeded; Lambda alias `url-shortener-dev-resolve-link:live` đang ở version `4`.
- Canary IAM: permissions boundary đã cho phép Lambda/SNS control-plane actions trên `Resource "*"`; trust policy và `iam:PassRole` vẫn giới hạn theo deployment group/service.
- Frontend: publish thành công tại [https://npt-shortenlink.dev](https://npt-shortenlink.dev); CloudFront route, callback, immutable asset và missing-object smoke checks đều pass.
- Alarms `resolve-errors`, `resolve-throttles`, `resolve-http-5xx`, `event-delivery-dlq`: trạng thái hiện tại `OK`.
- Kiểm tra cuối: Go race/vet/module verify pass; frontend lint/typecheck/build và 32 tests pass; Terraform bootstrap 24 tests và dev 22 tests pass; deployment Python 11 tests và redirect load tests 3 pass.

## Nợ kỹ thuật còn lại

| Mức | Mục | Bằng chứng hiện tại | Hướng xử lý |
|---|---|---|---|
| P2 | Dashboard tải toàn bộ danh sách và duy trì một socket cho mỗi link | `frontend/src/lib/api.ts:listAllLinks`, `useRealtimeClicks` | Phân trang server-side, một subscription theo owner, giới hạn fan-out và đo p95/p99 |
| P2 | Chưa có quota theo owner và kiểm soát abuse ở tầng ứng dụng | Terraform có throttling API nhưng chưa có quota/cost guard theo user | Quota DynamoDB/API usage, rate limit theo subject, cảnh báo chi phí và abuse |
| P2 | `Idempotency-Key` mới được gửi từ frontend, backend chưa lưu/đọc key cho POST create | `frontend/src/lib/api.ts` có header; backend create handler/service không xử lý key | Lưu kết quả create theo owner+key với TTL và kiểm thử retry |
| P2 | Phiên hết hạn chỉ redirect; chưa có refresh single-flight và retry có kiểm soát | Dashboard xử lý 401 bằng redirect; Cognito có refresh token nhưng chưa có manager chung | Xây session refresh manager, chống refresh đua nhau, giữ `returnTo` |
| P2 | Privacy consent mới có bằng chứng phía trình duyệt | `frontend/src/lib/auth.ts` lưu consent trong session/local storage; chưa có server-side evidence | Ghi policy version, timestamp, subject và purpose ở backend với retention rõ ràng |
| P2 | Counter analytics vẫn có nguy cơ hot key khi một link nổi bật | Aggregator ghi marker/counter theo link; chưa có sharding hoặc benchmark tải lớn | Shard counter theo time bucket, batch/write strategy và load test có ngưỡng |
| P2 | Frontend release chưa có rollback pointer/manifest bất biến | `publish-frontend.sh` cập nhật các key S3 mutable; CloudFront chỉ có cache invalidation | Publish theo release prefix, manifest pointer và quy trình rollback tự động |
| P3 | Chưa có schema/OpenAPI làm hợp đồng độc lập | Contract hiện nằm rải trong Go types, TS guards và Terraform tests | Sinh OpenAPI/JSON Schema từ nguồn chuẩn, kiểm tra drift trong CI |
| P3 | Dependency/tracing chưa có kế hoạch bảo trì dài hạn | CI đã có audit gate; OTel migration và lịch nâng cấp chưa được lập | Định kỳ nâng dependency, SBOM/advisory review và chuẩn hóa tracing |

Các mục P1 về WebSocket authorization, canary observability, wildcard deployment scope, URL validation, moderation workflow và delivery DLQ đã được xử lý trong đợt remediation này. Canary IAM mismatch là phát hiện mới trong lúc deploy; đã sửa boundary, thêm managed service policy và test regression.

## Nợ hạ tầng còn lại

| Mức | Mục | Trạng thái/bằng chứng | Hướng xử lý |
|---|---|---|---|
| P1 vận hành | Bootstrap IAM boundary tách khỏi Terraform dev deployment | Drift boundary từng làm canary fail; hiện đã apply bootstrap riêng và xác nhận AWS policy | Đưa bootstrap change vào release runbook/pipeline có version pin và preflight policy check |
| P2 | DNS custom domain chưa nằm trong Terraform | CloudFront alias + ACM certificate đã quản lý; không có Route 53 record/module | Quản lý Route 53 hoặc ghi rõ DNS owner, TTL, health check và drift check |
| P2 | S3 frontend không bật versioning/manifest release | Bucket có private/OAC nhưng module không có `aws_s3_bucket_versioning`; publish dùng key mutable | Bật versioning, release prefix, lifecycle và rollback manifest |
| P2 | Chưa có budget/cost alarm trong IaC | Có CloudWatch operational alarms và DLQ nhưng không thấy AWS Budgets/cost anomaly resources | Thêm budget theo account/service và alarm chi phí Lambda, CloudFront, DynamoDB, SQS |
| P2 | DLQ đã có nhưng chưa có replay/runbook | `EventDeliveryDLQ` và alarm đã deploy; chưa có script/automation replay và tiêu chí đóng incident | Viết runbook, replay tool an toàn, metric age/count và drill định kỳ |
| P3 | State backend khỏe nhưng bootstrap state local còn dễ nhầm | Dev dùng S3 state + lock; bootstrap đang giữ `terraform.tfstate` local | Đưa bootstrap state vào backend riêng hoặc quy định lưu trữ/backup rõ ràng |
| P3 | Secret hygiene của file tfvars local | `terraform.tfvars.json` bị ignore và không commit; vẫn là secret plaintext trên máy dev | Dùng secret manager/CI environment, rotate nếu file từng bị lộ, pre-commit secret scan |

## Ưu tiên thực hiện

1. Gần hạn: tự động hóa preflight boundary/bootstrap và thêm budget + DLQ replay runbook.
2. Trước khi tăng tải: owner quota/abuse control, idempotency server-side, pagination/owner subscription và frontend rollback.
3. Trong chu kỳ bảo trì: OpenAPI/schema, dependency cadence, tracing và cleanup bootstrap state.


