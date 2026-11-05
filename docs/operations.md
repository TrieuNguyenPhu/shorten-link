# Vận hành

## Triển khai

1. Build Lambda bằng `bash scripts/build-lambdas.sh`.
2. Chạy Terraform bootstrap để tạo state bucket, lockfile và OIDC deployment role.
3. Cấu hình `backend.hcl`, `terraform.tfvars` và secret Google OAuth qua biến môi trường.
4. Chạy Terraform environment `dev`.
5. Build frontend static, đồng bộ `frontend/out` lên S3 và invalidate CloudFront sau khi module hosting hoàn thiện.

## Kiểm tra sau deploy

- Gọi `GET /{code}` với link đã tạo và xác nhận redirect.
- Tạo/list/delete link bằng token Cognito Google.
- Kiểm tra EventBridge, SQS/DLQ, Lambda errors và DynamoDB counters.
- Mở WebSocket endpoint, subscribe theo code và xác nhận click update.
- Kiểm tra Step Functions execution cho URL bị từ chối.

## Xử lý sự cố

Xem CloudWatch log theo Lambda, age và DLQ depth của queue, Lambda throttles, API Gateway 4xx/5xx, Step Functions failures và CloudFront 4xx/5xx. Giữ lại bản ghi lỗi, xác định nguyên nhân rồi redrive có kiểm soát.

## Phát hành frontend

`bash scripts/publish-frontend.sh` đọc public endpoints từ Terraform outputs, build và kiểm tra `out/`, rồi đồng bộ S3 và invalidate CloudFront. Nút Google dùng callback của origin đã build; `.env.local` dành cho local, không commit. Trước rollout giữ bản sao các object mutable; script giữ lại bundle hash cũ để tab đang mở tiếp tục hoạt động.

Kiểm hosting sau deploy:

```sh
node tests/e2e/frontend-hosting.mjs https://d2aamyg2mt28wl.cloudfront.net
```

Kiểm auth thành công bằng Google thật vẫn cần phiên của người dùng. Regression callback dùng React DOM thật và fixture tại token endpoint, kiểm state, nonce, PKCE và redirect `/`. Không đưa fixture token vào website production.

## WebSocket và moderation

`useRealtimeClicks` đăng ký theo code; HTTP snapshot khi mở/reconnect giữ nguyên bảng và subscriptions, chỉ hiện loading khi tải lần đầu. Counter hiển thị không lùi khi snapshot cũ tới sau push; snapshot trễ không khôi phục code đã xóa thành công trong phiên dashboard. WebSocket là best-effort; refresh HTTP đối soát counter sau reconnect.

State machine moderation là workflow manual. Chỉ chạy với META đầy đủ đã tạo và input target khớp; không xóa link song song vì ASL `UpdateItem` hiện có thể upsert. Khi kết quả StartExecution không rõ, đối soát execution ARN đã đăng ký trước request; chỉ dọn META sau trạng thái terminal. Các regression cleanup nằm trong `tests/e2e/realtime-lifecycle.test.ps1`.

Click marker có TTL 24 giờ, DLQ giữ 14 ngày. Đối soát tuổi event/counter trước redrive vì marker đã mất có thể khiến replay đếm lại. Không purge queue hoặc redrive dữ liệu không thuộc phiên kiểm thử.

## Cổng kiểm tra trước phát hành

Chạy Go race/vet, DynamoDB Local integration, build tám ZIP; frontend lint/typecheck/build/test/audit; Terraform fmt/init/validate/test và PowerShell preflight. Terraform mock tests chạy trong checkout sạch không có `terraform.tfvars*` của AWS thật để các test về API chưa pin không bị cấu hình local ghi đè. Tham khảo [hướng dẫn local](local-testing.md).
