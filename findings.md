# Findings & Technical Decisions

## 1. Kết Quả Kiểm Toán Bản Gốc (`nghiadaulau/serverless-url-shortener-aws` & 21 Bài Viết)
- **Mốc nguồn:** ngày 2026-10-26, commit [`a5ffed9780d0447fc75a42def3693732bdc87907`](https://github.com/nghiadaulau/serverless-url-shortener-aws/tree/a5ffed9780d0447fc75a42def3693732bdc87907), 21 file được Git theo dõi. Đã đọc các file triển khai; lockfile được dùng làm metadata dependency, không chạy code tham chiếu.
- **Phạm vi bài viết:** ba lượt đọc toàn thân bài 1–7, 8–14, 15–21, theo liên kết trên cả hai trang series. Không còn bài chưa truy cập được. URL và đối chiếu từng bài nằm trong [reference audit](docs/phase-0-reference-audit.md).
- **8 Lambda Functions trong `template.yaml`:**
  1. `CreateLinkFunction` (`POST /links`, Cognito JWT)
  2. `ResolveLinkFunction` (`GET /{code}`, Public, alias `live` + CodeDeploy `Canary10Percent5Minutes`)
  3. `ListLinksFunction` (`GET /links`, Cognito JWT)
  4. `DeleteLinkFunction` (`DELETE /links/{code}`, Cognito JWT)
  5. `ClickAggregatorFunction` (SQS `ClickQueue`, `BatchSize: 10`, `ReportBatchItemFailures`)
  6. `WsConnectFunction` (WebSocket `$connect`)
  7. `WsDisconnectFunction` (WebSocket `$disconnect`)
  8. `ModerateCheckFunction` (Step Functions `CheckSafety`)
- **DynamoDB Single-Table (`url-shortener`):**
  - Khóa chính: `PK` (`HASH`), `SK` (`RANGE`).
  - Index `GSI1`: `GSI1PK` (`HASH`), `GSI1SK` (`RANGE`), `ProjectionType: ALL`.
  - Sparse Index: Chỉ `LINK#<code>/META` (`GSI1PK=USER#<ownerId>`, `GSI1SK=LINK#<createdAt>`) và `CONN#<id>/META` (`GSI1PK=WSCODE#<code>`, `GSI1SK=CONN#<id>`) xuất hiện trong `GSI1`. Các item `STAT#<YYYY-MM-DD>` và `CLICK#<eventId>` không có khóa `GSI1` nên tự động bị loại khỏi index.
  - TTL bật trên thuộc tính `ttl`: `CLICK#<eventId>` có TTL +24h (`86400` giây), `CONN#<id>` có TTL +2h (`7200` giây).
- **Điểm cần sửa so với code TypeScript gốc:**
  - Trong `click-aggregator.ts`, lệnh `catch` bắt mọi `TransactionCanceledException` và coi là duplicate. Bản Go phân loại đủ lý do hủy theo thứ tự META/STAT/marker, kiểm tra nil/độ dài và không nuốt conflict hoặc lỗi hỗn hợp; quy tắc chuẩn nằm trong [hợp đồng kiến trúc](docs/architecture-and-contracts.md).
  - Tách xử lý lỗi `pushClickUpdate` ra khỏi luồng lỗi của SQS record để lỗi WebSocket không bao giờ gây retry hoặc ảnh hưởng tới kết quả đếm click.
  - Source gốc nuốt lỗi `PostToConnection` ngoài 410; lỗi đọc/query/xóa connection ở DynamoDB mới có thể thoát lên aggregator. Push không có bảo đảm giao lại sau khi transaction đã commit.
  - Resolve phải đọc `FailedEntryCount` và kết quả từng entry của `PutEvents`, không coi HTTP 200 là đã publish thành công. Retry có giới hạn; lỗi cuối được báo thành API 5xx, không redirect thành công giả.
  - META Update cần điều kiện tồn tại để event đến muộn không tạo lại link đã xóa. Event xác nhận thuộc link đã mất được ghi log/metric và bỏ có chủ đích; lỗi chưa xác định vẫn retry.
  - Giữ TTL marker 24 giờ nhưng không gọi đây là exactly-once vô thời hạn. Replay sau khi marker bị xóa có thể đếm lại; cần đối soát trước redrive DLQ cũ.

### Những giới hạn giữ nguyên và khác biệt đã xác minh
- Redirect 301 có thể được browser cache; analytics đếm event được xử lý, không phải mọi lượt mở hay người dùng duy nhất. Publish lại có thể được EventBridge cấp ID mới.
- `$connect` công khai: ai biết code đều có thể nghe số click. List/delete HTTP vẫn enforce owner qua JWT và DynamoDB.
- Moderation độc lập, `MarkError` là Fail, không update status lỗi; không có quét URL thật. Activate/Reject của ASL gốc có thể upsert item, nên runbook phải chuẩn bị link hợp lệ trước khi chạy demo.
- Bài 21 còn mô tả dashboard như hướng mở rộng nhưng source cuối đã có `web/index.html`; chưa có S3/CloudFront hosting. UI đích không mang password demo hay nội suy `innerHTML` từ URL người dùng.
- Người dùng cập nhật ngày 2026-10-27: chỉ đăng nhập Google qua Cognito User Pool; giao diện Hand-Drawn / Doodle & Sketch UI. Đây là thay đổi so với source gốc. Không tạo password/signup flow, kể cả E2E; hai tài khoản Google thử nghiệm đi qua browser federation.
- Google OAuth Web client đặt redirect URI ở Cognito `/oauth2/idpresponse`; app callback ở CloudFront `/auth/callback/`. Cognito phát hành access token cho API Gateway JWT authorizer (ba route riêng tư yêu cầu scope `openid`), Lambda lấy Cognito `sub`. Google ID token/secret không vào frontend hoặc API. Google app/consent cần tạo ngoài AWS; Terraform quản lý Cognito IdP và state phải được bảo vệ vì có thể chứa Google secret. Nguồn: [AWS Google social IdP](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-user-pools-social-idp-google.html), [AWS HTTP API JWT authorizer](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api-jwt-authorizer.html).
- CI gốc chỉ build/validate; OIDC deploy là phần bổ sung. Custom bus/publisher cần có từ Phase 2; rule/SQS/aggregator nối hoàn chỉnh ở Phase 3.
- Số đo Node, quota tài khoản tác giả và hóa đơn lịch sử không phải benchmark/quota/chi phí bảo đảm của bản Go.

## 2. Trạng Thái Hạ Tầng AWS Hiện Tại
- Người dùng xác nhận ngày 2026-10-26 đã xóa toàn bộ tài nguyên AWS triển khai trước đó và làm lại từ đầu.
- Phiên hoàn thiện Phase 0 không chạy AWS CLI kiểm kê, apply/destroy hay load test; không xác nhận độc lập số lượng tài nguyên hoặc hóa đơn. Các con số 64/5 trong nhật ký cũ là thông tin lịch sử, không phải bằng chứng mới.
- Kế hoạch mới bootstrap remote state/OIDC bằng quyền operator riêng khi được phép, rồi mới dùng deployment role của ứng dụng; không dùng tài nguyên hay credentials của stack cũ.

## 3. Hệ Sinh Thái 13 Bộ Skills
- Kiểm tra filesystem ngày 2026-10-26: 314 thư mục cấp đầu có `SKILL.md`, 376 file `SKILL.md` nếu đếm đệ quy (bao gồm bản lồng/trùng); không cộng cả hai số thành skills độc lập.
- Đủ file đại diện của 13 nhóm: `using-superpowers`, `ponytail`, `planning-with-files`, `caveman`, `differential-review`, `react-best-practices`, `frontend-design`, `gsd-help`, `understand`, `open-code-review`, `ui-ux-pro-max`, `taste-skill`, `vibe-to-agentic-framework`.
- `.agents/skills.json` và `~/.gemini/config/skills.json` đều có entry `~/.agents/skills`. Việc file/config tồn tại không chứng minh mọi host tự nạp tất cả skills, hoặc nguồn gốc/phiên bản của từng bản cài đã được kiểm toán. Không cài lại khi hoàn thiện Phase 0.
