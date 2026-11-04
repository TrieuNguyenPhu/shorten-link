# Event pipeline — Phase 3 / Task 3.2

> Từ Task4.2, dev root có cả WebSocket/Step Functions và yêu cầu `moderation_boundary_arn` từ bootstrap. Trước khi dùng các lệnh triển khai dưới đây với code hiện tại, đọc [thứ tự bootstrap/app và pin API ID](./terraform-realtime-and-moderation.md#bootstrap-iam-và-thứ-tự-triển-khai-khi-được-yêu-cầu). Chưa có authorization apply mới.


## Phạm vi và trạng thái

Module `infrastructure/terraform/modules/event_pipeline` nhận name/ARN của custom bus do Task2.2 tạo; không tạo bus thứ hai. Root dev nối table/bus/boundary/package thật, thêm aggregator vào outputs roles/aliases và giữ bốn HTTP targets hiện có. Implementation và kiểm thử local hoàn tất; rule/SQS/ESM chưa được apply lên AWS trong task này.

Luồng: HTTP resolve → `url-shortener-events` → rule `source=urlshortener`, `detail-type=link.clicked` → ClickQueue → alias `live` của click-aggregator → DynamoDB transaction. Target giữ nguyên envelope, đặc biệt `id` dùng dedupe; không có input transformer hoặc ESM filter làm mất poison event.

## Cấu hình và quyền

- Queue chuẩn `${name_prefix}-ClickQueue`: visibility30giây, retention4ngày, redrive sau3lượt nhận. Giữ maxReceiveCount3 theo đặc tả, dù AWS khuyến nghị ít nhất5.
- `${name_prefix}-ClickDLQ`: retention14ngày, redrive allow chỉ source ARN ClickQueue. Cả hai dùng SSE-SQS và deny non-TLS; không thêm KMS key/quyền KMS.
- Queue policy chỉ cho principal `events.amazonaws.com` gửi message từ đúng rule ARN/account. Target được tạo sau policy.
- Aggregator dùng module Lambda chung: provided.al2023, arm64, bootstrap, memory128MB/timeout5s, Active tracing, Logs14ngày và alias live theo published version. ESM batch10, batching window0, `ReportBatchItemFailures`, enabled; chờ execution policy/alias và redrive allow policy.
- Execution policy: `dynamodb:UpdateItem`, `PutItem`, `GetItem` trên đúng table; `sqs:ReceiveMessage`, `DeleteMessage`, `GetQueueAttributes` trên đúng ClickQueue. `TransactWriteItems` không phải tên IAM action; transaction hiện tại không có thao tác ConditionCheck độc lập. Không cấp Query/GSI, WebSocket hoặc quyền consume DLQ cho runtime ở Phase3.
- Bootstrap boundary bổ sung UpdateItem và poll đúng queue, role allowlist chỉ năm handler đã có. App roles vẫn bị boundary và policy riêng theo từng hàm giới hạn.

Nguồn: [AWS SQS ESM](https://docs.aws.amazon.com/lambda/latest/dg/services-sqs-configure.html), [partial batch response](https://docs.aws.amazon.com/lambda/latest/dg/services-sqs-errorhandling.html), [EventBridge resource policies](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-use-resource-based.html), [DynamoDB SAR](https://servicereference.us-east-1.amazonaws.com/v1/dynamodb/dynamodb.json). SSE-SQS tương thích EventBridge; ví dụ trong [repository chính thức AWS](https://github.com/aws/aws-node-termination-handler/blob/main/README.md) dùng `SqsManagedSseEnabled=true` cùng EventBridge rules. Không nhầm với SSE-KMS dùng key mà EventBridge không được phép truy cập.

## Bootstrap, quota và pin ESM

Đợt triển khai đầu tiên hoặc thay thế ESM cần operator. Đây vẫn là tài nguyên Terraform quản lý, không tạo thủ công ngoài state. Sau khi được yêu cầu triển khai:

1. Operator review/apply bootstrap với source mới để cập nhật runtime boundary, năm role/function ARNs và managed policy pipeline.
2. Operator review/apply dev: tạo aggregator, queues, rule/target/policies và ESM. Deployment role chưa có quyền tạo ESM.
3. Lấy output `click_event_source_mapping_id`, đặt `click_event_source_mapping_ids` trong input bootstrap local, rồi operator review/apply bootstrap để pin UUID trước CD.
4. Deployment role chỉ Get/Update/Delete đúng ESM UUID và `lambda:FunctionArn` đúng alias aggregator live; ListTags/Tag/Untag cũng chỉ đúng UUID. Deny thay Project/Environment bằng mọi kiểu viết hoa/thường. Nếu plan thay thế ESM, operator provision và pin UUID mới; không mở wildcard TagResource để role tự nhận mapping khác.

ESM UUID chưa biết trước; wildcard tag quyền tạo/claim mapping không đáp ứng least privilege. Pattern operator provision rồi pin giống Cognito ở Phase2. Mọi thao tác apply trên đây cần yêu cầu triển khai rõ ràng; phiên Task3.2 chỉ chạy validate/mock plan.

Policy pipeline nằm tại `bootstrap/event_pipeline.tf`, là managed policy bootstrap sở hữu, được attach vào deployment role. Regression từng phát hiện tổng inline policy11256byte vượt10240; tách policy khắc phục mà không mở wildcard actions/resources. Tests kiểm tổng inline<=10240 và managed policy<=6144. App deployment role không được sửa bootstrap, managed policy này hoặc boundary của nó.

Autopilot đã thử nhưng `uvx` chưa có trong host. Theo fallback aws-iam, đối chiếu action/resource/condition qua SAR JSON chính thức cho Lambda/DynamoDB/SQS/EventBridge; không gọi account AWS. Command tái tạo baseline runtime khi công cụ khả dụng:

```sh
uvx iam-policy-autopilot@latest generate-policies \
  /home/trieunguyen/src/shorten-link/backend/internal/repository/dynamodb.go \
  /home/trieunguyen/src/shorten-link/backend/internal/repository/aggregator.go \
  /home/trieunguyen/src/shorten-link/backend/internal/events/eventbridge.go \
  --region ap-southeast-1 --account 654822777701 \
  --service-hints dynamodb events --pretty
```

Không upload policy bằng lệnh trên. [Lambda SAR](https://servicereference.us-east-1.amazonaws.com/v1/lambda/lambda.json) xác nhận Get/Update/DeleteEventSourceMapping hỗ trợ ESM ARN và `lambda:FunctionArn`; tagging cần scope ESM ARN riêng.

## Kiểm thử local và CI

```sh
bash scripts/build-lambdas.sh
terraform fmt -check -recursive infrastructure/terraform
terraform -chdir=infrastructure/terraform/bootstrap init -backend=false -input=false -lockfile=readonly
terraform -chdir=infrastructure/terraform/bootstrap validate
terraform -chdir=infrastructure/terraform/bootstrap test
terraform -chdir=infrastructure/terraform/environments/dev init -backend=false -input=false -lockfile=readonly
terraform -chdir=infrastructure/terraform/environments/dev validate
terraform -chdir=infrastructure/terraform/environments/dev test
pwsh -NoProfile -File tests/e2e/aws-flow.test.ps1
```

Terraform1.17.1/provider6.71.0:14bootstrap +13dev plan/mock tests PASS; không AWS apply trong tests. CI chạy toàn bộ file tests mới cùng parser/preflight Pipeline. Go1.26.10 race/vet và build5ZIP PASS; không đổi backend trong Task3.2. PowerShell7.5 container `--network none` kiểm invalid outputs/cross-account/cross-region/missing token, parser, deletion wait và unrelated scalar/array DLQ JSON; không gửi request AWS. Kiểm transaction và local HTTP analytics E2E xem [local-testing.md](./local-testing.md).

## AWS Pipeline E2E — chưa chạy

Sau deployment được phép, export outputs dev vào file local ignored, cấp `E2E_ACCESS_TOKEN_A` qua environment từ một tài khoản Google đã federation Cognito. Không dùng password flow. Chọn môi trường test yên tĩnh, ClickDLQ trống và operator CLI credentials đúng account có quyền đọc cấu hình, đọc table, PutEvents và consume/delete/change visibility trong DLQ:

```powershell
pwsh -NoProfile -File tests/e2e/aws-flow.ps1 -Scenario Pipeline -OutputsPath tests/e2e/dev.local.json
```

Script validate tất cả API/table/bus/queue/alias/UUID outputs cùng account/region trước token hoặc AWS, rồi kiểm operator STS account, rule/target, ESM Enabled/batch10/partial response, queue visibility30/redrive3 và DLQ retention14ngày. Nó tạo URL duy nhất qua API thật, gọi GET đúng1lần không follow301 và consistent-read chờ `META.clicks=1`, tổng STAT theo UTC=1.

Tiếp theo, script PutEvents một `link.clicked` với code vừa tạo và timestamp không hợp lệ để buộc consumer retry; giữ nguyên EventId được ack và chờ message đó vào DLQ, tối đa40lượt poll. Ghi EventId, SQS MessageId và ApproximateReceiveCount quan sát được ở DLQ. Không suy diễn receive count của DLQ là số lần consumer nhận tại source queue; bằng chứng là redrive config3 thực đã đọc và poison qua ESM đến DLQ. Kiểm poison không tăng counter hoặc tạo marker, rồi chỉ xóa receipt của đúng event test. Message không liên quan được trả visibility0; JSON scalar/array không làm scenario dừng sớm.

Cleanup chỉ delete code có target/shortUrl khớp URL test vừa tạo, rồi bounded wait404. Không purge queue, đổi visibility config, tắt ESM, sửa IAM hoặc redrive dữ liệu. Nếu fail sau khi publish poison, event có thể tiếp tục đến DLQ sau cleanup: dùng EventId đã log để kiểm tra/dọn message test khi được phép, không xóa message khác.

## Giới hạn và redrive

Marker24giờ ngắn hơn DLQ14ngày; phải đối soát tuổi event/nguyên nhân/counter trước replay. Marker còn dù TTL quá hạn vẫn dedupe; marker mất rồi replay có thể đếm lại. Không redrive toàn bộ backlog cũ. SQS có thể duplicate delivery; publish lại sau transport timeout có EventBridge ID mới. Không hứa exactly-once vô thời hạn.

ClickDLQ là DLQ của consumer SQS, không phải DLQ cho lỗi EventBridge→queue delivery; target hiện dùng retry mặc định, chưa có target delivery DLQ riêng theo kiến trúc nguồn. AWS event delivery/ESM/IAM/redrive vẫn pending cho tới khi chạy scenario thật. WebSocket notifier/endpoint/quyền ManageConnections thuộc Phase4; Phase3 chưa có realtime push.
