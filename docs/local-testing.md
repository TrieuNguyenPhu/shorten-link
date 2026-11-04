# Kiểm thử core HTTP API trên máy local

Task 2.1 chạy handler/service/repository thật với DynamoDB Local. Không dùng kho link trong bộ nhớ. Unit tests thay dependency tại ranh giới repository/SDK/publisher; integration và HTTP E2E dùng DynamoDB Local. Recorder chỉ thay dịch vụ EventBridge tại ranh giới giao thức HTTP, không phát sinh số click giả và không chuyển sự kiện đến consumer.

## 1. Khởi động DynamoDB Local

Yêu cầu Go 1.26.10, GCC/CGO cho race detector và Docker. Từ root repository:

```sh
docker run --rm -d --name shorten-link-ddb \
  -p 127.0.0.1:8000:8000 \
  amazon/dynamodb-local:3.3.0@sha256:d89f8fcc6b1a39cb35976c248ed42a28c66ae00dc043099210f5571e42648ab4 \
  -jar DynamoDBLocal.jar -inMemory -sharedDb
```

Chờ log báo khởi động xong (`docker logs shorten-link-ddb`). Bảng local dùng PK/SK, GSI1PK/GSI1SK và projection ALL đúng hợp đồng. `-inMemory` làm dữ liệu local mất khi container dừng; đây vẫn là engine DynamoDB Local thực thi conditions/query, không phải repository in-memory trong ứng dụng. [Hướng dẫn AWS](https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/DynamoDBLocal.DownloadingAndRunning.html).

## 2. Các cổng kiểm tra

```sh
go -C backend test -race -count=1 ./...
go -C backend vet ./...
DYNAMODB_ENDPOINT=http://127.0.0.1:8000 \
  go -C backend test -race -tags=integration -count=1 -v \
  ./internal/repository/... ./tools/local-http/...
bash scripts/build-lambdas.sh
```

Hoặc dùng `DYNAMODB_ENDPOINT=http://127.0.0.1:8000 make test-integration`. Không bỏ qua integration khi thiếu endpoint: test báo lỗi rõ ràng. Test từ chối endpoint ngoài loopback, dùng credentials cố định `local/local`, tạo bảng tên duy nhất và xóa riêng bảng đó khi kết thúc. Không đọc credential chain của operator.

Integration repository kiểm collision bằng conditional PutItem, chống IDOR bằng conditional DeleteItem, đọc GetItem, list đúng owner, timestamp trong cùng giây và pagination cursor thực với 6 item khoảng 220 KB/item (vượt giới hạn 1 MB/trang). Delete chỉ xóa META, giữ STAT và CLICK marker.

HTTP E2E dựng server adapter thật, chạy bốn route qua net/http và SDK v2; kiểm unauthenticated 401, JSON null/nonstring/URL sai 400, create 201, danh sách theo tenant, delete khác owner/missing cùng 404, redirect 301 có Location/body rỗng, publish bị từ chối trả 502 và link đã xóa trả 404. Redirect client tắt tự follow.

## 3. HTTP harness cho thao tác thủ công

Từ root, mở terminal riêng:

```sh
go -C backend run ./tools/local-http \
  -config ../scripts/local-http/config.example.json \
  -init-table -record-events
```

Cấu hình mẫu nằm ở [scripts/local-http/config.example.json](../scripts/local-http/config.example.json): listener 127.0.0.1:8080, DynamoDB 127.0.0.1:8000, endpoint PutEvents 127.0.0.1:8010, bảng `url-shortener-local`, bus `local-events`, identities `alice` và `bob`. Có thể sao chép cấu hình sang file local; listener và hai dependency endpoints bắt buộc là HTTP/IP loopback, không credentials/query/fragment trong URL. Không dùng config/credentials AWS thật.

`-init-table` tạo bảng local nếu chưa có. `-record-events` bật recorder tại endpoint trong config: nhận một PutEvents entry, kiểm schema, ghi `localRecordedClick` và trả ID xác nhận cục bộ. Đây là fixture opt-in, không phải EventBridge hay mô phỏng analytics. Bỏ cờ này nếu dùng endpoint test bên ngoài; recorder không tồn tại trong bốn Lambda production. Không có recorder hoạt động thì resolve trả 5xx khi publish thất bại.

Header `X-Local-User: alice` hoặc `bob` chỉ được adapter local ánh xạ thành claims `sub` thử nghiệm. Danh tính khác/thiếu header bị coi là chưa đăng nhập; bearer JWT tự gửi vào harness không được xác thực. Production chỉ đọc `requestContext.authorizer.jwt.claims.sub` do API Gateway cung cấp, không tin header local hay ownerId trong body.

```sh
curl -i -X POST http://127.0.0.1:8080/links \
  -H 'X-Local-User: alice' -H 'Content-Type: application/json' \
  -d '{"url":"https://example.org/path"}'
curl -i http://127.0.0.1:8080/links -H 'X-Local-User: alice'
curl -i http://127.0.0.1:8080/links -H 'X-Local-User: bob'
```

Lấy `code` từ response create, rồi thay `<code>` bên dưới:

```sh
curl -i -X DELETE 'http://127.0.0.1:8080/links/<code>' -H 'X-Local-User: bob'
curl -i 'http://127.0.0.1:8080/<code>'
curl -i -X DELETE 'http://127.0.0.1:8080/links/<code>' -H 'X-Local-User: alice'
curl -i 'http://127.0.0.1:8080/<code>'
```

Các status tương ứng: 404, 301, 200, 404. Không thêm `curl -L` khi kiểm 301. `shortUrl` ở harness luôn trỏ địa chỉ local được cấu hình; Lambda dùng `PUBLIC_REDIRECT_BASE_URL`, hoặc HTTPS domainName API Gateway khi biến này không đặt. Ctrl+C dừng harness; `docker stop shorten-link-ddb` dọn container local.

## 4. Deadline, lỗi publish và giới hạn bằng chứng

Request production được giới hạn 4 giây, thấp hơn timeout Lambda 5 giây. Publisher dùng deadline tối đa 2 giây (giữ deadline cha nếu ngắn hơn), tắt retry SDK riêng cho PutEvents, retry entry `InternalFailure`/`ThrottlingException` tối đa 3 lần gọi tổng cộng với full jitter 0–50 ms và 0–100 ms. HTTP 200 chỉ thành công khi FailedEntryCount=0, đúng một entry có EventId không rỗng và không có lỗi. Transport error có kết quả bất định nên trả lỗi ngay; không publish lại tự động ở tầng ứng dụng. [AWS PutEventsResultEntry](https://docs.aws.amazon.com/eventbridge/latest/APIReference/API_PutEventsResultEntry.html), [SDK retries/timeouts](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html).

Resolve missing phát LinkNotFound; publish lỗi phát ClickPublishFailed, trả error từ Lambda (adapter local ánh xạ 502); chỉ publish được chấp nhận mới phát LinkResolved và trả 301. Lỗi metric được log nhưng không làm publish lại. Moderation status không chặn redirect theo đặc tả hiện tại.

Body create tối đa 16 KiB sau giải mã base64, phải là một object JSON với URL string hợp lệ. List giữ bốn trường item code/target/clicks/createdAt, trả trang theo limit/cursor và optional nextCursor; client theo cursor đến hết nếu cần toàn bộ danh sách. GSI eventual consistency và browser cache 301 vẫn áp dụng.

Kết quả local không chứng minh Cognito Google federation/access-token scopes, JWT authorizer, IAM, trace export, bus/rule delivery, SQS/ESM/DLQ hoặc canary. AWS E2E giữ pending cho đến khi được phép triển khai; không chạy terraform apply hoặc load test AWS trong Task 2.1.

## 5. Bằng chứng ngày 2026-11-04

Go 1.26.10/Linux amd64 và DynamoDB Local 3.3.0 (digest bên trên): unit/race, vet, mod verify, repository integration và HTTP E2E đều exit 0. Lệnh integration dùng `-race -tags=integration -count=1 -v` như mục 2; hai test TestDynamoDBLocalCRUDIsolationAndPagination và TestLocalHTTPEndToEnd PASS. Artifact build nằm tại `backend/dist/{create-link,resolve-link,list-links,delete-link}.zip` (ignored, không commit); mỗi ZIP chứa bootstrap Linux arm64. CI thêm cùng integration/E2E và build bốn archive. Chưa có bằng chứng AWS deployment.


## 6. Cursor pagination sau final review

`GET /links?limit=25&cursor=<nextCursor>`: limit1–100, mặc định25, cursor trả400 nếu malformed hoặc thuộc owner khác. Khi còn dữ liệu, response có nextCursor; trang cuối chỉ có links. Mỗi trang được giới hạn byte budget trước JSON Lambda envelope; không silently truncate. Local HTTP harness chuyển query parameters thật vào handler. Unit regression duyệt450link với URL dài/nhiều ký tự cần escape, kiểm tất cả response<6MiB và không mất/trùng code. Integration kiểm limit1 qua2 trang, hết cursor và cross-owner400 bằng DynamoDB Local.
