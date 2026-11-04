# Terraform bootstrap và core infrastructure — Task 2.2

Task 2.2 cung cấp cấu hình triển khai, không phải bằng chứng đã triển khai AWS. Task 2.1 đã merge qua PR #71; root `bootstrap` và `environments/dev` được kiểm bằng Terraform plan với mock provider. Không có AWS apply, JWT federation, IAM hoặc pipeline service test trong phiên này.

## 1. Cấu trúc và kiểm tra local

- `infrastructure/terraform/bootstrap`: S3 state private/versioning/AES256/TLS, GitHub OIDC/deployment role và boundary cho execution roles. State bootstrap độc lập, mặc định local; deployment role không truy cập hoặc sửa bootstrap.
- `infrastructure/terraform/environments/dev`: S3 backend có encrypt/use_lockfile, DynamoDB, Cognito, custom event bus, bốn Lambda và HTTP API.
- Modules `dynamodb`, `cognito`, `lambda`, `http_api`, `event_bus` có version constraint; hai root pin Terraform **1.17.1** và AWS provider **6.71.0**, commit `.terraform.lock.hcl`.

Từ root repository, chạy:

```sh
make terraform-check
```

Target build bốn ZIP thật trước khi Terraform tính source_code_hash, rồi chạy:

```sh
terraform fmt -check -recursive infrastructure/terraform
terraform -chdir=infrastructure/terraform/bootstrap init -backend=false -input=false -lockfile=readonly
terraform -chdir=infrastructure/terraform/bootstrap validate
terraform -chdir=infrastructure/terraform/bootstrap test
terraform -chdir=infrastructure/terraform/environments/dev init -backend=false -input=false -lockfile=readonly
terraform -chdir=infrastructure/terraform/environments/dev validate
terraform -chdir=infrastructure/terraform/environments/dev test
```

Test chỉ dùng `command = plan`, `mock_provider`, computed fixtures và input thử nghiệm; không có test apply hoặc credential AWS. `override_during = plan` làm ARN fixture có sẵn cho assertion JSON IAM. DynamoDB GSI dùng `key_schema` theo provider 6.71.0; primary key vẫn PK/SK, GSI1 vẫn GSI1PK/GSI1SK, projection ALL và ttl.

CI thêm các kiểm tra này và PowerShell parser/preflight. `make check` hiện gồm backend unit/race/vet và Terraform; integration DynamoDB Local vẫn chạy riêng bằng `make test-integration` khi có container.

## 2. Chuẩn bị trước khi triển khai được phép

Cần operator credentials/SSO cho tài khoản đích, `aws_account_id` đúng 12 chữ số, bucket name toàn cục chưa trùng và prefix Cognito duy nhất. Region mặc định `ap-southeast-1`; provider dùng allowed_account_ids để tránh nhầm tài khoản. Thay các placeholder trong `.example` bằng giá trị thật ở file local được ignored, không dùng input test của `.tftest.hcl` để deploy.

Đối với Google OAuth Web client, tạo/configure ngoài AWS:

1. Chọn prefix Cognito trước khi deploy. Authorized redirect URI của Google phải là `https://<prefix>.auth.ap-southeast-1.amazoncognito.com/oauth2/idpresponse`.
2. Cho phép hai tài khoản Google thử nghiệm trong consent screen nếu app còn ở testing mode.
3. Truyền client ID vào input Terraform; truyền client secret bằng `TF_VAR_google_client_secret` từ secret store/input bảo mật, không ghi secret vào argument, shell history, `.tfvars.example`, PR hoặc log.
4. Cognito app callback khác Google redirect: local là `http://localhost:3000/auth/callback/`; sau Task 5.3 dùng URL CloudFront HTTPS thật và cập nhật callback/logout/CORS chính xác.

Cognito chỉ hỗ trợ Google ở public app client, grant `code`, scopes `openid email profile`, token revocation, access/ID token15phút, refresh1ngày. Chỉ bật `ALLOW_REFRESH_TOKEN_AUTH`, không bật password/SRP/custom auth; self-signup native bị tắt. Lite/classic domain version1 phục vụ redirect trực tiếp Google, không thêm managed-login branding hoặc tier trả phí. PKCE S256/state/nonce do frontend Task 5.1 gửi và kiểm; Terraform client không có cờ bắt buộc PKCE riêng. Không có password form hoặc signup flow riêng.

## 3. Bootstrap bằng operator, tách ứng dụng

Các bước dưới đây là runbook cho lần triển khai được người dùng yêu cầu rõ ràng, chưa được thực thi trong task này.

- Tạo bản local từ `bootstrap/terraform.tfvars.example`; đặt `umask 077`, bảo vệ state/backup trên storage được mã hóa. Bootstrap mặc định dùng local backend độc lập, không đưa state này cho deployment role ứng dụng. Backup an toàn sau mỗi thay đổi; đừng commit state, backup hay plan.
- Nếu tài khoản đã có GitHub OIDC provider, import resource `aws_iam_openid_connect_provider.github` bằng ARN provider thật trước plan; không tạo duplicate. Không pin thumbprint GitHub cũ vì IAM dùng trusted CA store cho provider này.
- Review bootstrap plan bằng operator. Bucket có prevent_destroy và không force_destroy; versioning/AES256/public block/TLS deny đều khai báo rõ. Không thêm DynamoDB lock table: S3 dùng native `.tflock`.
- Sau apply được phép, ghi nhận outputs bucket, deployment_role_arn, runtime_boundary_arn và github_subject. Bootstrap role/policy nằm ở path `/url-shortener/bootstrap/`, execution roles ở `/url-shortener/dev/`.

Trust mặc định đúng subject `repo:TrieuNguyenPhu/shorten-link:environment:dev` và aud `sts.amazonaws.com`. GitHub environment `dev` cần protection/reviewers và deployment branch policy chỉ main. Nếu chủ động chọn `github_environment = null`, subject chuyển thành exact `repo:TrieuNguyenPhu/shorten-link:ref:refs/heads/main`; không đồng thời cho cả hai hay dùng wildcard. Workflow CD sẽ được nối ở Task 6.2, dùng manual dispatch, serialized concurrency và OIDC; CI task này không có id-token/write hoặc deploy.

State role chỉ GetObject/PutObject trên key ứng dụng, thêm DeleteObject cho đúng `.tflock`; ListBucket giới hạn prefix `dev/*`. Backend dành workspace metadata dưới `dev/workspaces`, nhưng object permissions chỉ hỗ trợ state mặc định; không dùng Terraform workspaces để tạo môi trường khác. Không có quyền xóa state, đổi bucket policy, provider trust hoặc boundary. State bucket/versioning có thể chứa Google secret trong state cũ: hạn chế quyền đọc và bảo vệ cả lịch sử phiên bản. `sensitive = true` chỉ che display, không loại secret khỏi state/plan. [HashiCorp S3 backend](https://developer.hashicorp.com/terraform/language/backend/s3).

Deployment role có quyền CRUD đúng loại tài nguyên app; Cognito pool được operator provision lần đầu và pin cognito_user_pool_ids trước CD. Deployment role không được CreateUserPool hoặc gắn tag để nhận quyền trên pool khác. Mỗi execution role cần boundary bootstrap; PassRole chỉ bốn role đã xác định và chỉ cho `lambda.amazonaws.com`. Role có thể quản lý inline policy ứng dụng nhưng boundary chặn quyền ngoài DynamoDB/bus/logs/X-Ray hiện tại; không được bỏ/sửa boundary hoặc cập nhật chính deployment role. Khi thêm aggregator/WebSocket/SFN ở phase sau, phải mở rộng bootstrap boundary/deployment permissions có chủ đích, không cấp trước quyền SQS/Step Functions/ManageConnections.

## 4. Scope IAM và vòng đời API

- Create chỉ PutItem bảng; resolve GetItem bảng và PutEvents đúng bus; list Query đúng GSI1 ARN; delete DeleteItem bảng. Mỗi hàm chỉ ghi log group riêng; log group do Terraform tạo trước Lambda, retention14ngày. Active tracing và hai quyền ghi X-Ray có Resource `*` vì không hỗ trợ resource scope.
- Deployment dùng explicit action lists, role/function names, table/bus ARN, account/region, Cognito exact pool ARNs sau operator provisioning. Read-only DescribeLogGroups và DescribeUserPoolDomain cần Resource `*`; domain read còn bị giới hạn RequestedRegion.
- API Gateway v2 không được giả định kế thừa tags từ API xuống route/integration/authorizer. Quyền child CRUD giới hạn bốn loại path trong region. Khi chưa có ID, `http_api_ids = []` dùng wildcard ở đúng vị trí API ID để bootstrap lần đầu; điều này cho phép thao tác child của API khác trong cùng region, nên chỉ dùng cho operator/bootstrap window.
- Sau lần app apply được phép đầu tiên, lấy `http_api_id`, đặt `http_api_ids = ["<id-thật>"]` trong bootstrap và review/apply cập nhật bằng operator **trước khi bật CD**. Tag endpoints cũng scope theo ID (gồm dạng URL-encoded), không dùng điều kiện ResourceTag trên endpoint không hỗ trợ. Deny ngăn đổi/xóa Project/Environment qua tag endpoints bằng so sánh TagKeys không phân biệt hoa/thường. API replacement cần operator cập nhật allowlist ID tương ứng.

Giới hạn ID là bước vận hành bắt buộc trước CD; mock tests xác minh pin ID tạo đúng ARN, không thay bằng bằng chứng STS/IAM thật. Nguồn: [AWS tagged resources và giới hạn inheritance V1](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-tagging-supported-resources.html), [AWS Service Authorization Reference](https://servicereference.us-east-1.amazonaws.com/).

Đã thử IAM Policy Autopilot nhưng host không có uvx/pip/ensurepip; các cách cài qua venv/pip đều không chạy được. Theo fallback của skill aws-iam, đã đối chiếu tên action/resource capabilities qua Service Authorization Reference thay vì suy đoán action theo tên SDK. Khi có công cụ, có thể tái tạo baseline từ runtime bằng lệnh sau rồi review scope theo từng hàm (không upload policy):

```sh
uvx iam-policy-autopilot@latest generate-policies \
  /home/trieunguyen/src/shorten-link/backend/internal/repository/dynamodb.go \
  /home/trieunguyen/src/shorten-link/backend/internal/events/eventbridge.go \
  --region ap-southeast-1 --service-hints dynamodb events \
  --tf-dir /home/trieunguyen/src/shorten-link/infrastructure/terraform/environments/dev --pretty
```

## 5. App plan và alias live

Build `backend/dist/*.zip` trước Terraform. Root dev dùng backend S3; copy `backend.hcl.example` thành `backend.hcl`, điền bucket output bootstrap và giữ encrypt/use_lockfile. Init với backend config thật chỉ khi được phép kết nối account; `init -backend=false` chỉ dành validate/mock tests.

Inputs app gồm account ID, boundary ARN, Google client ID/secret, prefix Cognito và exact callback/logout/CORS. Các URL localhost trong ví dụ là callback dev hợp lệ, không phải endpoint API giả. API tạo `$default` stage auto-deploy, throttle10RPS/burst20, JSON access log không ghi token/URL body. Ba owner routes dùng JWT issuer Cognito/client ID, scope `openid`; public `GET /{code}` không authorizer.

Environment được truyền theo hàm: TABLE_NAME cho cả bốn; EVENT_BUS đúng bus thật cho resolve; PUBLIC_REDIRECT_BASE_URL từ API endpoint cho create, không có CloudFront domain ở Phase2. Custom bus có ngay Phase2; chưa có rule/SQS/aggregator nên redirect thành công không chứng minh click đã được tổng hợp.

Mỗi Lambda publish version và có alias live; integrations và invoke permissions đều dùng alias, permission có source_account và route/method ARN. Resolve alias có ignore_changes function_version/routing_config để CodeDeploy làm writer duy nhất sau này. Những lần Terraform publish version mới chưa tự đổi resolve alias; trước Task6.1/6.2, rollout code resolve cần quy trình canary được hoàn thiện, không tự reset alias để né quy tắc. Ba alias private vẫn theo published version mới bình thường.

Deletion protection mặc định bật cho DynamoDB và Cognito, PITR bật cho bảng. Dọn AWS cần yêu cầu riêng; khi được phép, review việc tắt deletion protection trước destroy ứng dụng. Giữ state bucket và bootstrap độc lập; không force-delete phiên bản state hoặc bucket để dọn nhanh.

## 6. AWS E2E Core — vẫn pending

Sau khi app được phép triển khai, export outputs không có secret ra file local ignored:

```sh
terraform -chdir=infrastructure/terraform/environments/dev output -json > tests/e2e/dev.local.json
```

Dùng hai tài khoản Google đi qua browser → Cognito authorization code + PKCE để lấy access token A/B và Cognito ID token A. Chuẩn bị riêng Google ID token cho negative test qua OAuth testing tool/process được phép; không thêm Google token flow vào frontend. Cấp qua environment E2E_ACCESS_TOKEN_A, E2E_ACCESS_TOKEN_B, E2E_ID_TOKEN_A và E2E_GOOGLE_ID_TOKEN bằng secret input; không in token hoặc dùng command arguments. Chạy khi được phép:

```powershell
pwsh -NoProfile -File tests/e2e/aws-flow.ps1 -Scenario Core -OutputsPath tests/e2e/dev.local.json
```

Script kiểm endpoint API Gateway HTTPS, token preflight và hai sub khác nhau trước HTTP; API Gateway thực hiện xác minh chữ ký/issuer/scope thật. Kiểm thiếu JWT401, ID/Google token bị từ chối401/403, invalid/null/nonstring400, access token đúng scope200/201, A list/B không thấy và B delete A404 giống missing, 301 với redirect disabled. Cleanup chỉ code có target/shortUrl khớp link vừa tạo, trong finally; kiểm resolve404 sau xóa. Output evidence chỉ check/status, không token/sub. Đây là smoke nhỏ, không phải load test.

Kiểm local đã chạy parser PowerShell7.5 và negative endpoint preflight, không gửi request AWS. STS trust negative tests bằng OIDC token thật, Google browser federation, IAM, X-Ray export và pipeline rule/SQS giữ pending; các scenario Pipeline/Canary/Moderation sẽ được bổ sung ở task sở hữu. Không tuyên bố các scenario chưa có đã chạy.


## 7. AWS E2E OIDC — script sẵn sàng, thực thi vẫn pending

`aws-flow.ps1 -Scenario OIDC` dùng outputs của bootstrap (deployment_role_arn/github_subject), không dùng outputs ứng dụng. Export thành `tests/e2e/bootstrap.local.json` khi deployment đã được phép. Cần năm JWT GitHub Actions thật, chưa hết hạn, qua input bảo mật: E2E_OIDC_ALLOWED_TOKEN, E2E_OIDC_WRONG_AUDIENCE_TOKEN, E2E_OIDC_WRONG_SUBJECT_TOKEN, E2E_OIDC_WRONG_REF_TOKEN, E2E_OIDC_FORK_TOKEN. Không dùng JWT tự ký hoặc sửa payload để thay token thật; không log, commit hay lưu token vào artifact công khai.

Allowed token phải đúng configured sub/aud. Wrong audience giữ sub đúng nhưng aud khác STS; wrong subject là cùng repo nhưng environment/PR subject khác; wrong ref là branch khác main không gắn protected environment; fork token thuộc repo fork khác. Với environment trust, branch không nằm trong sub nên GitHub environment branch protection phải được kiểm riêng; token của PR từ fork trên base repo được kiểm dưới wrong-subject `pull_request`.

```powershell
pwsh -NoProfile -File tests/e2e/aws-flow.ps1 -Scenario OIDC -OutputsPath tests/e2e/bootstrap.local.json
```

Script kiểm tất cả issuer/expiry/aud/sub trước kết nối, gọi STS AssumeRoleWithWebIdentity bằng POST HTTPS, session900giây. Allowed phải200; wrong audience phải InvalidIdentityToken, các subject/ref/fork sai phải AccessDenied. Không deserialize, ghi hoặc dùng credentials trả về; chỉ xuất check/status. Kiểm parser/role-path/missing-token preflight đã chạy local; chưa có STS execution evidence. Workflow thu thập token và vận hành sandbox được nối đầy đủ ở Task6.2 khi được phép.


## 8. Final review fixes và lần apply được phép

Người dùng đã yêu cầu sửa lỗi và apply sau final review ngày2026-11-04. Triển khai bằng operator trước, sau đó pin cả http_api_ids/cognito_user_pool_ids và cập nhật bootstrap trước khi role CD được dùng. CreateUserPool không được cấp cho deployment role; quyền Tag/Untag pool chỉ trên ARN được pin và bị deny ownership keys. Thêm quyền đọc GetFunctionCodeSigningConfig cho các function ARN hiện có.

AWS Core kiểm delete bằng Wait-DeletedLink (tối đa10 reads, cách250ms, chỉ retry301; status khác fail ngay), phù hợp eventual reads. API list dùng cursor contract đã được người dùng chốt, xem architecture3.2. Saved plan/state/input OAuth phải được bảo vệ, không commit hoặc log secret. State/resource output thực sẽ ghi trong PROJECT_STATUS sau apply thành công; không coi fixture/mock test là bằng chứng AWS.


## 9. Bằng chứng triển khai 2026-11-04

Theo yêu cầu apply rõ ràng của người dùng, account654822777701/ap-southeast-1 đã nhận10bootstrap +42core resources; policy deployment sau đó update1lần để pin API txnaexyh3d và pool ap-southeast-1_SGj2otrc0. Không có destroy. Endpoint https://txnaexyh3d.execute-api.ap-southeast-1.amazonaws.com . Cấu hình OAuth đọc trực tiếp từ JSON local, callback hiện http://localhost:3000/auth/callback/ trước Phase5.

AWS smoke đã xác minh JWT negatives, CRUD/list cursor qua operator Lambda invokes, HTTP301/404, publisher ack và redirect Cognito→Google; đã xóa cả2link thử. Đây chưa phải Core E2E với Google access tokens thật. Bootstrap state backup riêng tại /home/trieunguyen/.local/state/shorten-link/bootstrap-terraform.tfstate, quyền0600; inputs/state/plan không commit.

Cognito tự thêm username mapping và Google endpoint metadata khi read-back. Module ignore đúng các entries tự sinh, giữ client_id/client_secret/authorize_scopes và email/name dưới quản lý Terraform; [source provider6.71.0](https://github.com/hashicorp/terraform-provider-aws/blob/v6.71.0/internal/service/cognitoidp/identity_provider.go) đọc toàn bộ maps. Final bootstrap/core plan đều detailed-exitcode0. IAM Access Analyzer trên policy đã pin không có errors/security warnings. CI170 của PR#73 PASS cả4jobs trước commit chốt evidence/default handling.
