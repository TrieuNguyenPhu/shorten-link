# Findings & Technical Decisions

## 1. Kết Quả Kiểm Toán Bản Gốc (`nghiadaulau/serverless-url-shortener-aws` & 21 Bài Viết)
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
  - Trong `click-aggregator.ts`, lệnh `catch` bắt mọi `TransactionCanceledException` và coi là duplicate. Trong Go, ta kiểm tra `CancellationReasons`: chỉ bỏ qua khi phần tử thứ 3 (`Put CLICK#<id>`) trả về `ConditionalCheckFailed`.
  - Tách xử lý lỗi `pushClickUpdate` ra khỏi luồng lỗi của SQS record để lỗi WebSocket không bao giờ gây retry hoặc ảnh hưởng tới kết quả đếm click.

## 2. Trạng Thái Hạ Tầng AWS Hiện Tại
- Đã chạy `terraform destroy` xóa hoàn toàn 64 tài nguyên trong stack `dev` cũ và 5 tài nguyên trong stack `bootstrap` cũ.
- Đã kiểm tra bằng AWS CLI: không còn S3 bucket, DynamoDB table, Lambda function, API Gateway, Cognito User Pool, hay CloudFront distribution nào tồn tại.

## 3. Hệ Sinh Thái 13 Bộ Skills
- Đã cài đặt 265 skills từ 13 repositories vào `C:\Users\NguyenPhuTrieu\.agents\skills` (tổng cộng 314 skills) và đăng ký trong `~/.gemini/config/skills.json` cũng như `.agents/skills.json` tại workspace.
