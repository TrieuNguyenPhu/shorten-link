# WebSocket và moderation — Task 4.1

## Phạm vi và cấu hình

Task4.1 cung cấp ba Lambda Go `ws-connect`, `ws-disconnect`, `moderate-check`, repository connection, pusher cho aggregator và [ASL moderation](../statemachine/moderation.asl.json). Build bằng `bash scripts/build-lambdas.sh` tạo đủ tám ZIP Linux arm64, mỗi archive chỉ chứa `bootstrap` quyền0755.

Hai Lambda WebSocket cần `TABLE_NAME`. Aggregator nhận thêm `WS_ENDPOINT` dạng management URL HTTPS `https://<api-id>.execute-api.<region>.amazonaws.com/prod`, không dùng `wss://` hoặc thêm `/@connections`. Endpoint được lấy từ cấu hình triển khai, không lấy từ input click. Khi chưa đặt biến này, aggregator chỉ ghi counter như Phase3. Task4.2 mới tạo WebSocket API/stage, IAM và truyền endpoint. `moderate-check` chỉ kiểm tra input, không cần bảng hoặc AWS credentials.

Runtime DynamoDB/Management API dùng AWS SDK v2, retry tối đa2attempts/backoff100ms và X-Ray instrumentation như các Lambda hiện có. Hai handler WebSocket giới hạn4giây; notifier giữ budget200ms sau commit trong deadline4giây của batch. Không đủ thời gian thì push có thể dở dang, chỉ log lỗi; không redeliver record đã commit để cố gửi lại notification.

## Hợp đồng WebSocket

Kết nối `wss://<api-id>.execute-api.<region>.amazonaws.com/prod?code=<code>` là public theo kiến trúc5.4: người biết code đều có thể theo dõi count. Không kiểm owner/token hay sự tồn tại của link. Thiếu/rỗng/trắng `code` hoặc thiếu connectionId trả400; lưu lỗi trả500. `$connect` ghi:

```json
{
  "PK": "CONN#<connectionId>",
  "SK": "META",
  "code": "Abc1234",
  "GSI1PK": "WSCODE#Abc1234",
  "GSI1SK": "CONN#<connectionId>",
  "ttl": 1793800800
}
```

`ttl` thực tế là Unix timestamp lúc xử lý+7200giây, không phải giá trị cố định trong ví dụ. `$disconnect` chỉ cần connectionId và xóa trực tiếp `CONN#id/META`, gọi lặp vẫn trả200. TTL được DynamoDB dọn bất đồng bộ, không phải cơ chế xác thực.

Sau `ClickApplied`, pusher đọc `LINK#code/META.clicks` với `ConsistentRead=true`, query mọi trang `GSI1PK=WSCODE#code`, rồi gọi `PostToConnection` với JSON:

```json
{"code":"Abc1234","clicks":1}
```

Chỉ `GoneException` (HTTP410) làm xóa connection. Lỗi post khác được log và tiếp tục peer sau; lỗi Get/Query/cleanup cũng được log qua aggregator, không fail record SQS đã commit. Duplicate/deleted không push. GSI vẫn eventual consistency; việc kết nối hoặc disconnect vừa diễn ra có thể chưa phản ánh trong query. Delivery best-effort, không replay; frontend Phase5 phải nạp snapshot khi reconnect và hợp nhất count bằng `max` theo đặc tả.

[AWS Management API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-how-to-call-websocket-api-connections.html) xác định endpoint/stage, SigV4 và `GoneException`. Pusher sử dụng SDK ký request, không gọi endpoint ẩn danh.

## Moderation và ASL

Input trực tiếp của Lambda và workflow:

```json
{"code":"Abc1234","target":"https://example.org/docs"}
```

Output Lambda giữ nguyên `code`, `target`, thêm `safe`. `target` chứa một trong `malware`, `phishing`, `evil` sau khi chuyển chữ thường thì `safe=false`. Đây là quy tắc substring của bản gốc, không gọi dịch vụ quét URL. Thiếu/rỗng/trắng code hoặc target làm Lambda trả lỗi; không ghi log target.

ASL giữ hai placeholder `${ModerateCheckArn}` và `${TableName}` cho Terraform Task4.2. `CheckSafety.Resource` nhận trực tiếp Lambda ARN/alias, nên output là `{code,target,safe}`, không có lớp `Payload`; [AWS direct Lambda integration](https://docs.aws.amazon.com/step-functions/latest/dg/connect-lambda.html) mô tả hình thức này. Retry `Lambda.ServiceException`, `Lambda.TooManyRequestsException`, `States.TaskFailed`: IntervalSeconds1, MaxAttempts3, BackoffRate2. Catch `States.ALL` tới `MarkError` kiểu `Fail`, Error=`ModerationFailed`, không ghi `status=error`.

`IsSafe` dẫn tới `Activate` (`status=active`) hoặc `Reject` (`status=rejected`) bằng `dynamodb:updateItem`. Giữ nguyên hành vi không có ConditionExpression: item không tồn tại sẽ bị upsert thành META thiếu dữ liệu. Create không đặt pending hoặc StartExecution; redirect không xét status. Link rejected vẫn redirect được theo hợp đồng này.

## Runbook manual AWS — chưa chạy trong Task4.1

Chỉ chạy phần này sau khi Task4.2 đã được review, triển khai theo yêu cầu người dùng và có outputs endpoint, table, state machine ARN. Trong Task4.1 không apply hoặc gọi AWS. Tài liệu [kiểm thử local](./local-testing.md) mô tả bằng chứng hiện có.

1. Tạo hai link thật bằng `POST /links` với Cognito access token: target `https://example.org/docs` (safe) và `https://example.org/evil` (reject). Giữ code trả về cho từng link; không tự chọn code rồi để workflow tạo META. Với mỗi code, dùng `GetItem --consistent-read` kiểm `LINK#code/META` có target đúng, ownerId, createdAt và clicks. Giữ nguyên link trong toàn bộ execution, không xóa song song với moderation.
2. Chạy một execution cho mỗi input `{code,target}` đúng link tương ứng. Theo dõi history: `CheckSafety` → `IsSafe` → `Activate`/`Reject`; execution `SUCCEEDED`, đọc META consistent để xác nhận active/rejected. Kiểm ownerId/createdAt/target/clicks không đổi. Gọi redirect để kiểm hợp đồng status không chặn301; thao tác redirect sẽ tạo click thật.
3. Kiểm retry/Fail bằng link safe vẫn tồn tại và input `{code:<code-safe>,target:""}`. Handler validation lỗi, `States.TaskFailed` được retry3lần sau lần gọi đầu, thời gian chờ1/2/4giây (chưa tính invocation). History phải có tổng4lần Lambda invocation và kết thúc `MarkError`, execution `FAILED` với `ModerationFailed`. META giữ status cũ, không có `error` status. Đây là lỗi input có chủ đích tại ranh giới Lambda thật, không thêm handler giả hoặc thay đổi code triển khai.
4. Kiểm WebSocket: mở client với `?code=` thiếu giá trị phải fail handshake400; mở hai client cho code safe và một client cho code reject. Redirect code safe một lần (không lặp qua browser cache301); chờ aggregator xử lý và so payload với META consistent. Chỉ subscriber đúng code nhận count. Đóng client rồi kiểm item mất khi `$disconnect` được giao; đóng lặp không tạo lỗi. Các tình huống `410` stale connection/non410 được chứng minh ở HTTP fixture local; AWS fault injection cần kịch bản riêng được cho phép, không suy ra đã kiểm từ đóng socket bình thường.
5. Lưu execution ARN/history, counter trước/sau, response status và log lỗi cần thiết làm bằng chứng. Sau khi execution kết thúc, đóng client và xóa hai link thử bằng API của đúng owner. Không xóa hàng loạt STAT/CLICK hoặc redrive dữ liệu cũ.

Ví dụ CLI sau khi operator đã điền input thực tế và xác nhận META tồn tại:

```sh
aws dynamodb get-item --table-name "$TABLE_NAME" \
  --key "$(jq -cn --arg code "$CODE" '{PK:{S:("LINK#"+$code)},SK:{S:"META"}}')" \
  --consistent-read

aws stepfunctions start-execution --state-machine-arn "$STATE_MACHINE_ARN" \
  --input "$(jq -cn --arg code "$CODE" --arg target "$TARGET" '{code:$code,target:$target}')"

aws stepfunctions describe-execution --execution-arn "$EXECUTION_ARN"
aws stepfunctions get-execution-history --execution-arn "$EXECUTION_ARN"
```

`CODE`/`TARGET` phải khớp link đã tạo. Với kịch bản Fail, chỉ đặt TARGET rỗng sau khi đã xác minh META hợp lệ; không xóa link để tạo lỗi. Execution history chứa target nên dùng URL ví dụ không có token hoặc dữ liệu cá nhân. History Lambda thực tế và WebSocket handshake/IAM/ESM trên AWS đều còn pending.
