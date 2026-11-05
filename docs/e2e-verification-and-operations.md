# Phase 6 — CI/CD, kiểm chứng và vận hành

## Phạm vi và trạng thái

Task 6.1 ở [PR #89](https://github.com/TrieuNguyenPhu/shorten-link/pull/89); merge trước Task 6.2. Hai task có branch riêng từ main. Task 6.2 không chép lại module của Task 6.1. Workflow từ chối chạy khi thiếu module canary hoặc thiếu outputs của hạ tầng đã provision.

Bằng chứng local: Go race/vet/build tám ZIP; DynamoDB Local repository/HTTP/WebSocket integration; Terraform mock-plan; frontend lint/typecheck/build/27 tests; release failure-path tests và k6 smoke local. Chưa apply Phase 6, chạy workflow CD, assume OIDC bằng workflow mới, xác nhận SNS, canary/rollback hoặc load test AWS. Google federation bằng tài khoản thật vẫn pending. Không coi fixture CLI/PutEvents local là bằng chứng dịch vụ AWS.

## Cấu hình trước khi bật CD

1. Operator review/apply bootstrap và app Phase 6.1 riêng **sau khi được người dùng cho phép**; quy trình hiện tại yêu cầu stack đã tồn tại, không bootstrap tài khoản sạch bằng CD. App deployment role không quản lý bootstrap.
2. Giữ các pin đã có: Cognito pool, HTTP/WebSocket APIs, click ESM UUID. Với Task 6.2, operator lấy IDs từ `terraform state show` các resource hosting hiện tại; nếu đã apply output mới thì dùng `terraform output -json frontend_resource_ids` từ app, truyền object đó vào `frontend_resource_ids` của bootstrap rồi operator review/apply. Object gồm distribution, OAC, hai cache policies và response headers policy. Khi chưa pin, role không có quyền sửa CloudFront. Replacement tài nguyên hosting cần operator provision/repin; CD không có quyền tạo distribution/bucket khác.
3. GitHub environment **dev** phải có required reviewers, hạn chế deployment chỉ main, chặn bypass nếu tổ chức hỗ trợ. Workflow chỉ `workflow_dispatch`, đúng repo/main, không PR/fork. Concurrency `url-shortener-dev-deployment` không cancel release đang chạy. Không chạy publisher hoặc Terraform writer khác song song.
4. Từ bootstrap outputs, đặt environment variables `AWS_DEPLOY_ROLE_ARN`, `AWS_ACCOUNT_ID`, `TF_STATE_BUCKET`, `RUNTIME_BOUNDARY_ARN`, `MODERATION_BOUNDARY_ARN`. Đặt `COGNITO_DOMAIN_PREFIX`, `GOOGLE_CLIENT_ID` theo deployment hiện tại; secret `GOOGLE_CLIENT_SECRET` ở environment secret. Prefix cố định theo default `url-shortener-dev`, region `ap-southeast-1`. Giữ các allowlists đã cấu hình bằng environment variables JSON arrays `CALLBACK_URLS`, `LOGOUT_URLS`, `CORS_ORIGINS` (mặc định `[]`, CloudFront vẫn được module thêm tự động). `ALARM_EMAIL_ADDRESSES` mặc định `[]`, chỉ đặt địa chỉ đã đồng ý nhận SNS; `LOG_RETENTION_DAYS` mặc định `14`. Đối chiếu với operator inputs trước review plan để tránh đổi cấu hình ngoài ý muốn. Không đặt secret vào `NEXT_PUBLIC_*` hoặc artifact.
5. Kiểm trust có exact audience `sts.amazonaws.com` và subject `repo:TrieuNguyenPhu/shorten-link:environment:dev`, khớp `github_subject` bootstrap. GitHub environment subject không chứa branch nên branch restriction của environment là bắt buộc. Nếu repository bật immutable subject IDs, operator cần đối chiếu claim thật và sửa trust trước; không nới wildcard. Không ghi JWT vào log.
6. Các actions được pin SHA, Go 1.26.10, Node 26.13.0, pnpm 10.33.2, Terraform 1.17.1/AWS provider 6.71.0. CI chạy offline mocks và DynamoDB Local; CD chạy `make check` trước assume role.

## Hai lần dispatch: duyệt plan rồi apply

1. Chọn workflow **Deploy dev**, branch main, action **plan**. Reviewer duyệt quyền chạy environment. Job kiểm các cổng local, build ZIP, assume role bằng OIDC, init remote state, kiểm CodeDeploy không có deployment active/alias weighted và tạo saved plan.
2. Job summary chỉ có resource addresses/actions. Plan nhị phân chứa dữ liệu nhạy cảm được SSE-S3 encrypt trong bucket state private, prefix `dev/plans/<run-id>/`; không upload vào GitHub artifact. `PutObject --if-none-match '*'` không ghi đè plan cùng run ID; rerun cùng plan ID thất bại, cần dispatch mới.
3. Reviewer có quyền operator tải plan về thư mục riêng (umask 077) để kiểm **chi tiết thay đổi redacted**, IAM và những resource có thể thay thế. Không chỉ dựa vào tên resource trong summary. Ví dụ sau chỉ đọc, dùng run ID đã kiểm:

   ```bash
   umask 077
   aws s3 cp "s3://$TF_STATE_BUCKET/dev/plans/$PLAN_RUN_ID/app.tfplan" /tmp/reviewed-app.tfplan
   terraform -chdir=infrastructure/terraform/environments/dev show -no-color /tmp/reviewed-app.tfplan
   rm /tmp/reviewed-app.tfplan
   ```

   Không dùng `show -json` để đẩy raw plan lên log, artifact hoặc chat. Binary có thể chứa Google secret dù CLI hiển thị redacted. Bucket versioning giữ bản cũ; lifecycle riêng xóa cả current/noncurrent plan sau hai ngày (xử lý bất đồng bộ), không ảnh hưởng state/version history.
4. Sau khi review, dispatch action **apply** cùng main commit, nhập **plan_run_id**. Approval environment là bước cuối sau review plan. Manifest buộc đúng repo, commit, hash ZIP và digest plan; main đã tiến lên hoặc binary khác thì tạo/review plan mới. Terraform tự từ chối stale state plan. Không tự plan lại trong apply.
5. Apply đúng binary đã duyệt → lấy outputs thật → build/test static frontend và validate public config bằng chính validator của browser → CodeDeploy canary → upload hashed assets → route data → HTML → invalidate/wait CloudFront → hosting smoke. Lỗi apply/build/canary/rollback chặn upload. Không xóa hashed assets cũ. Hash config được kiểm lại trước publish để tránh dùng artifact build với endpoint khác.
6. Canary lấy CurrentVersion từ alias live thật, TargetVersion từ published output. Trùng version là no-op. CodeDeploy giữ 10% trong năm phút rồi 100%, dừng khi error/throttle alarm hoặc không đọc được alarm. Terraform ignore version/routing của alias resolve. Script không gọi `update-alias`.
7. Script đợi deployment terminal; khi failure tiếp tục đợi ID và terminal state của rollback, kiểm alias baseline và hết active deployments, rồi vẫn trả lỗi để chặn frontend. Mất response CreateDeployment, timeout hoặc rollback chưa xác nhận: không retry mù, không reset alias; operator đối soát history trước release mới. Giới hạn chờ mặc định 1800s (tối đa 3600s); AWS CLI có connect/read timeout. Workflow có giới hạn 60 phút, nên timeout/cancel cũng cần đối soát.
8. Thành công xóa hai plan objects hiện tại; version cũ còn đến lifecycle expiry. Khi publish frontend lỗi sau backend success, backend vẫn ở version đã rollout; sửa lỗi rồi tạo plan mới, canary same-version sẽ no-op. Khôi phục frontend từ artifact đã xác minh bằng publisher sau khi được phép, không xóa bundle cũ.

## Ma trận E2E AWS được cấp phép

Dùng AWS sandbox/operator identity đúng account/region và outputs thật. Token Google/Cognito nhập qua environment, không tham số/log; có hai tài khoản Google thử nghiệm cho owner isolation. Snapshot outputs công khai lưu vào file `*.local.json` đã ignored, không dump state/tfvars/secret.

| Luồng | Lệnh / đầu vào | Bằng chứng cần lưu |
| --- | --- | --- |
| Auth/CRUD | `pwsh tests/e2e/aws-flow.ps1 -Scenario Core -OutputsPath tests/e2e/dev.local.json` | Thiếu JWT, Google/ID token bị chặn; access token đúng scope; A tạo/list, B không thấy/xóa A; 301 Location, invalid URL/body, cleanup |
| Bus/SQS/ESM/DLQ | `-Scenario Pipeline` | Event ID, queue/ESM ARN, counter, retry/receive count và DLQ; dùng các fault inputs/preconditions trong script; không dùng local thay bằng chứng ESM |
| WebSocket/SFN | `-Scenario Realtime` | Socket push count, disconnect/410 cleanup; moderation safe/reject và Retry/Catch/Fail bằng execution history; giữ META đến khi execution terminal |
| OIDC | `-Scenario OIDC` với **bootstrap outputs** | Năm GitHub-signed JWT thật: allowed, wrong audience, wrong subject, wrong ref, fork; STS allow/deny, không lưu JWT/session credentials |
| Hosting | `node tests/e2e/frontend-hosting.mjs "$FRONTEND_URL"` | Routes/callback/assets 200, immutable headers, missing routes/assets 403/404 |
| Google browser | Browser thật, Google → Cognito → CloudFront callback | state/nonce/PKCE, về `/`, API access token, reconnect/dashboard; không password flow, không fixture token production |
| Canary success/rollback | Các bước dưới đây | Deployment/rollback IDs, terminal statuses, alias trước/sau, alarm transition, API 301 sau rollout |

### Canary trong sandbox

Provision stack sandbox đã đổi prefix/account rõ ràng bằng Terraform khi được phép; SNS subscription rỗng để fault test không gửi email. Tạo link thật bằng API và giữ code cho lần gọi GET kiểm cuối; snapshot outputs có `resolve_published_version`, alias map và CodeDeploy names.

```bash
export AWS_REGION=ap-southeast-1
export E2E_CANARY_SANDBOX=1
export E2E_CANARY_ACCOUNT_ID='<authorized-sandbox-account>'
export E2E_REDIRECT_CODE='<existing-test-link-code>'
bash tests/e2e/canary.sh tests/e2e/sandbox.local.json success
# Hoặc từ PowerShell trên Linux/WSL:
# pwsh tests/e2e/aws-flow.ps1 -Scenario Canary -CanaryScenario success -OutputsPath tests/e2e/sandbox.local.json
```

Script chứng minh same-version no-op trước, rồi chạy target khác current, kiểm alias không weighted và real HTTP API 301 sau rollout. Để kiểm rollback, operator dùng **Terraform ZIP input trong sandbox** provision một published target cố ý trả Lambda error; giữ alias live ở version tốt (ignore_changes), không đổi code/role/config production. Dùng file outputs mới và chạy `canary.sh ... rollback`. Script chỉ invoke target sau khi thấy weighted routing của canary; bắt buộc `FunctionError`, chờ auto rollback hoàn tất và kiểm live về baseline, rồi gọi HTTP API thật. Không dùng SetAlarmState để giả lỗi ứng dụng, không publish frontend hoặc reset alias thủ công.

Lưu deployment ID từ script, lấy `aws deploy get-deployment --deployment-id <id>` và rollback ID; lấy `cloudwatch describe-alarm-history --alarm-name <resolve-error-alarm> --history-item-type StateUpdate` trong cửa sổ thử nghiệm để đối chiếu alarm stop. Kiểm HTTP integration của API trỏ ARN `resolve-link:live` bằng `apigatewayv2 get-integrations`, không `$LATEST`. Khôi phục ZIP tốt qua Terraform plan/apply sau test; xóa link test khi alias ổn định và execution liên quan đã terminal. Không xóa version đang được alias/deployment tham chiếu.

### k6

Dùng k6 2.3.0; container đã kiểm local: `grafana/k6:2.3.0@sha256:2dd4756ec6e14e62778916e1eb850108e4b499304c02df56760fa0bdb0fba0b8`.

```bash
# Chỉ chạy AWS khi được yêu cầu rõ; URL/code phải là link test thật.
k6 run -e MODE=smoke -e BASE_URL="$HTTP_API_ENDPOINT" -e CODE="$TEST_CODE" tests/load/redirect.k6.js
# Stress chỉ sau authorization riêng:
k6 run -e MODE=stress -e ALLOW_STRESS=1 -e BASE_URL="$HTTP_API_ENDPOINT" -e CODE="$TEST_CODE" tests/load/redirect.k6.js
```

Mặc định 2 VUs/10s, sleep 1s mỗi iteration (xấp xỉ tối đa 2 RPS), không follow redirects. Stress ramp tới 40 VUs, có thể chạm API throttle hiện tại; báo successful RPS, failed RPS/rate, 429 và 5xx riêng. Threshold đòi 301/Location hợp lệ, failed rate=0 nên stress có throttle sẽ exit nonzero. Không tự động chạy stress trong CI/CD; không dùng benchmark Node hoặc số liệu loopback để kết luận hiệu năng AWS/Go.

Phiên local 2026-11-05: Go HTTP handler/service thật + DynamoDB Local 3.3.0 + recorder PutEvents opt-in, 20/20 redirect 301, khoảng 1.99 successful RPS, zero failures/throttle. Recorder chỉ xác nhận protocol publish, không chuyển EventBridge/SQS, không tạo analytics delivery.

## Triage và redrive

- Resolve Errors/Throttles alarm: xem log request ID và X-Ray, kiểm DynamoDB/EventBridge/IAM và concurrency. Throttle HTTP API 429 khác Lambda Throttles; đối chiếu cả API response và dashboard. Alarm function-wide cũng thấy direct version invocation trong fault test.
- ClickPublishFailed: request có thể đã publish nhưng response mất; không coi mọi 5xx là event chưa tồn tại. Đối soát Event IDs/counter trước replay thủ công.
- Backlog/oldest-age tăng: kiểm EventBridge rule, SQS queue policy, ESM Enabled, IAM poll và `ReportBatchItemFailures`. Partial record failure có thể không tăng Lambda Errors; DLQ/backlog là tín hiệu riêng.
- DLQ > 0: lấy mẫu với `sqs receive-message` có visibility timeout ngắn, không xóa trước đối soát; xem tuổi event, source/detail-type/id/code/at và nguyên nhân record. ClickDiscardedDeleted thường là event đến sau delete, không tự coi duplicate.
- Dedupe TTL 24h, DynamoDB xóa bất đồng bộ; DLQ giữ 14d. Marker còn thì duplicate được ack; marker mất thì replay cùng event có thể đếm lại. Không redrive blanket dữ liệu cũ. Đối soát META/STAT/marker và event age, chia lô nhỏ, giữ nguyên EventBridge id/envelope, ghi quyết định rồi mới redrive subset đủ điều kiện khi được phép. Nếu chưa xác định tính đúng của count thì giữ message/evidence để xử lý, không sinh id mới để lách dedupe.
- SNS topic không subscription không có người nhận; chỉ bật địa chỉ đã đồng ý, người nhận phải confirm. Logs mặc định 14d, DLQ 14d, marker 24h, socket TTL 2h. Theo dõi chi phí CloudWatch alarms/dashboard/logs/traces, SQS, Lambda, CloudFront/S3 trước test và dọn dữ liệu phiên riêng; không dọn tài nguyên ngoài phiên.

Nguồn: [GitHub AWS OIDC](https://docs.github.com/en/actions/how-tos/secure-your-work/security-harden-deployments/oidc-in-aws), [AWS CodeDeploy AppSpec](https://docs.aws.amazon.com/codedeploy/latest/userguide/reference-appspec-file-structure-resources.html), [deployment status API](https://docs.aws.amazon.com/cli/latest/reference/deploy/list-deployments.html), [k6 redirect behavior](https://grafana.com/docs/k6/latest/using-k6/http-requests/), [CloudFront IAM SAR](https://servicereference.us-east-1.amazonaws.com/v1/cloudfront/cloudfront.json).
