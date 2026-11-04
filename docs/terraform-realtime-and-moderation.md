# Terraform WebSocket và Step Functions — Task4.2

## Tài nguyên và kết nối

`environments/dev` giữ bốn module Lambda HTTP hiện có, thêm ba Lambda `realtime_functions` (`ws-connect`, `ws-disconnect`, `moderate-check`), gọi module `websocket_api` và `moderation_sfn`. Aggregator vẫn thuộc `event_pipeline`, tổng cộng tám Lambda Go/arm64. Mọi integration dùng alias `live`; create/redirect không có quyền StartExecution.

`websocket_api` tạo API protocol WEBSOCKET, route selection `$request.body.action`, đúng hai route public `$connect`/`$disconnect`, stage `prod` auto-deploy, throttle10req/s và burst20. Lambda proxy giữ envelope WebSocket có `requestContext.connectionId`; không gửi payload HTTP2.0. Lambda invoke permission pin account, API, stage prod, route và qualifier live. Không tạo `$default`/`$send` hoặc authorizer vì contract hiện tại là subscription public theo code.

Outputs root:

| Output | Sử dụng |
| --- | --- |
| `websocket_api_id` | Pin bootstrap IAM sau operator provisioning |
| `websocket_endpoint` | `wss://<api>.execute-api.<region>.amazonaws.com/prod` cho browser |
| `websocket_management_endpoint` | HTTPS cùng API/stage, truyền thành `WS_ENDPOINT` của aggregator |
| `websocket_connections_arn` | `arn:aws:execute-api:<region>:<account>:<api>/prod/POST/@connections/*` |
| `moderation_state_machine_arn` | Manual StartExecution và execution history |
| `moderation_role_arn` | Role riêng chỉ cho Step Functions |
| `lambda_roles`, `lambda_aliases` | Map đủ tám handler |

Aggregator thêm Query trên đúng `table/index/GSI1`, DeleteItem trên bảng để cleanup connection và ManageConnections trên đúng POST/prod/API. Endpoint và ARN phải khớp API/region/account của bảng; module standalone vẫn cho bỏ cả hai input để giữ hành vi Phase3. [AWS Management API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-how-to-call-websocket-api-connections.html) mô tả HTTPS endpoint và SigV4; không truyền URL wss vào SDK.

`moderation_sfn` render trực tiếp [ASL Task4.1](../statemachine/moderation.asl.json) với alias `moderate-check:live` và bảng thật, type STANDARD. Service role dùng `states.amazonaws.com`, trust pin SourceAccount và SourceArn của đúng machine. ARN được tính từ tên/account/region để không tạo vòng phụ thuộc role↔state machine; xem [AWS confused deputy protection](https://docs.aws.amazon.com/step-functions/latest/dg/procedure-create-iam-role.html).

Runtime role và boundary moderation riêng chỉ cho InvokeFunction trên alias moderation và UpdateItem trên bảng. Workflow vẫn manual, không có rule kích hoạt tự động. `safe=false` chỉ ghi status=rejected, không chặn redirect. ASL giữ update không condition nên phải có META hợp lệ trước và không chạy song song delete; [runbook Task4.1](./websocket-and-moderation.md) mô tả safe/reject/retry/Fail.

Lambda có log group riêng, retention theo cấu hình và Active X-Ray như module nền. Task4.2 dùng execution history tích hợp của Standard; chưa bật CloudWatch execution logging/trace cho state machine hoặc logging account-level của WebSocket API. Việc bổ sung observability thuộc Phase6, cần quyền logging riêng; không tự quản lý API Gateway account shared hoặc cấp thêm quyền log cho role moderation chỉ có hai quyền nghiệp vụ.

## Bootstrap IAM và thứ tự triển khai khi được yêu cầu

Không chạy apply trong Task4.2. Bootstrap là root do operator quản lý, deployment role không được sửa bootstrap policy/boundaries. Trước đợt triển khai được người dùng cho phép:

1. Build đủ tám ZIP. Cập nhật bootstrap với tám Lambda trong allowlist và boundary riêng moderation; cấp output `moderation_boundary_arn` vào dev input cùng `runtime_boundary_arn`. Hai ARN phải đúng account/name_prefix. Giữ các ID HTTP API, Cognito và ESM đã pin; không bỏ ID cũ để có wildcard discovery.
2. Operator apply dev để tạo WebSocket API/machine và các Lambda mới. Deployment role chưa được tạo/claim child của WebSocket API chưa pin. Khi `websocket_api_ids=[]`, runtime boundary không cho ManageConnections; push chưa hoạt động dù role aggregator có policy cho API mới.
3. Lấy đúng output `websocket_api_id`, thêm vào `websocket_api_ids` của bootstrap rồi operator apply bootstrap. HTTP và WebSocket IDs được hợp nhất để scope API child/tag paths; boundary cho aggregator chỉ POST @connections trên prod của các API đã pin. Từ đây CD có thể bảo trì API đã có. Giữ ESM operator-first/pin của Phase3, không thay bằng wildcard.
4. Trước khi đưa deployment role vào CD, kiểm lại plan không có destroy ngoài ý muốn, outputs và mọi policy mới đã được apply. Không chỉ cập nhật app stack mà bỏ qua bootstrap.

Các grant quản lý Lambda/API chuyển từ inline sang managed policies `*-lambda-deploy` và `*-api-deploy`, cùng managed `*-realtime-deploy` mới. Regression test tái hiện inline12314byte vượt10240 trước khi tách; test cuối kiểm aggregate inline≤10240 và từng managed/boundary≤6144 khi pin đủ HTTP/WebSocket/Cognito/ESM. Quyền nghiệp vụ và ownership-tag protections được giữ khi di chuyển; không mở iam:CreatePolicyVersion hoặc quyền thay bootstrap.

Step Functions PassRole chỉ đúng role moderation với PassedToService=states.amazonaws.com; Lambda PassRole giữ service lambda.amazonaws.com và allowlist tám role. Role moderation không dùng chung boundary Lambda. Deployment chỉ quản lý machine theo tên cụ thể, không được StartExecution. Provider AWS6.71.0 còn cần `states:ListStateMachineVersions` khi refresh và `states:ValidateStateMachineDefinition` trong plan khi definition thay đổi. Validation API không hỗ trợ resource-level permission nên statement riêng dùng Resource=* với RequestedRegion cố định; không cấp `states:*`.

IAM policy generation skill đã thử Autopilot (`uvx`, `pip`) nhưng host không có các executable này; dùng Service Authorization Reference chính thức và source provider pin6.71.0 để đối chiếu action/resource. Không upload policy hoặc gọi AWS account trong phiên này.

## Kiểm thử offline và AWS E2E

```sh
GOTOOLCHAIN=go1.26.10 bash scripts/build-lambdas.sh
terraform fmt -check -recursive infrastructure/terraform
for root in infrastructure/terraform/bootstrap infrastructure/terraform/environments/dev; do
  terraform -chdir="$root" init -backend=false -input=false -lockfile=readonly
  terraform -chdir="$root" validate
  terraform -chdir="$root" test
 done

docker run --rm --network none -v "$PWD:/workspace" -w /workspace \
  mcr.microsoft.com/powershell:7.5-ubuntu-24.04 \
  pwsh -NoProfile -File tests/e2e/aws-flow.test.ps1
```

Terraform tests dùng mock provider tại plan, không tạo tài nguyên AWS. Bao gồm module wiring, native WebSocket envelope/live alias, output endpoints, aggregator permission, mismatch account, SFN ASL/trust/boundary, pin IDs và IAM quota. CI tự chạy toàn bộ tests và preflight Realtime cùng Core/OIDC/Pipeline.

Khi đã triển khai theo authorization riêng, xuất outputs vào file local ignored, giữ Cognito access token trong environment và chạy bằng operator credentials đúng account:

```powershell
terraform -chdir=infrastructure/terraform/environments/dev output -json > tests/e2e/dev.local.json
./tests/e2e/aws-flow.ps1 -Scenario Realtime -OutputsPath tests/e2e/dev.local.json
```

Cần `E2E_ACCESS_TOKEN_A` thật từ Cognito Google flow, có scope openid; không đặt token trong args hoặc output. Operator cần DescribeStateMachine/StartExecution/DescribeExecution/GetExecutionHistory/StopExecution, DynamoDB GetItem/Query, GetApi/GetStage, GetFunctionConfiguration và STS identity theo đúng tài nguyên. Các quyền vận hành này không tự cấp cho GitHub deployment role.

Scenario xác thực tất cả output/account/region trước token/network, kiểm GetApi của cả hai API bằng operator credentials trước khi gửi bearer token/mở socket, kiểm deployed ASL và WS_ENDPOINT; tạo hai link thật, kiểm META đầy đủ, chạy active/rejected, rồi input target rỗng để xác nhận4Lambda invocations và MarkError/ModerationFailed. Hai ClientWebSocket thật subscribe code safe, đợi GSI có đủ hai connection/TTL, redirect một lần và nhận count1 khớp consistent META. Polls/socket operations đều có budget. Cleanup đóng socket, dừng/chờ mọi execution đang chạy trước khi xóa các link do test tạo; giữ link và báo lỗi nếu không xác nhận được terminal state.

Đây là kịch bản đã viết, chưa chạy AWS trong Task4.2. CI chỉ parse và kiểm preflight/negative resource inputs trong container network=none. Không suy ra execution history, handshake/IAM, delivery pipeline, TTL deletion hoặc stale410 trên AWS đã PASS. `410`/non410 vẫn có bằng chứng local từ Task4.1; fault injection AWS cần kịch bản được cho phép riêng.
