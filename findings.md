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

## Phase 1 — Quyết định triển khai 2026-10-27
- Phạm vi Task 1.1: domain/config + test, Makefile, Go CI, tài liệu local; Task 1.2: logging/EMF/X-Ray và hai entrypoint build dùng Go ZIP helper. Chưa có caller nghiệp vụ; Phase 2 sẽ dùng các API này. Không thêm handler placeholder, Terraform hoặc sửa frontend untracked.
- Pre-flight: Task 2.1 tiêu thụ domain/config/middleware; giữ EVENT_BUS theo đặc tả, WS_ENDPOINT là HTTPS Management API, PublicRedirectBaseURL tùy chọn để handler có thể fallback domainName. TABLE_NAME bắt buộc; config chung không bắt mọi Lambda khai báo các endpoint không dùng.
- Ruling Git: tạo từng branch từ origin/main rồi fast-forward baseline phụ thuộc; PR 1.1 base nhánh Phase 0, PR 1.2 base nhánh 1.1. PR #65 đang OPEN; chưa merge main, tránh đưa lại toàn bộ reset Phase 0 vào diff task. Retarget sau khi upstream merge.
- Go local 1.26.5; pin toolchain 1.26.5 (không tuyên bố mới nhất), CI Linux chạy race. Windows chưa có gcc trong PATH, có WSL Ubuntu để kiểm chứng race.
- AWS xác nhận X-Ray SDK/daemon maintenance từ 25/02/2026, chỉ cập nhật bảo mật; lựa chọn instrumentation sẽ được ghi rõ ở Task 1.2. Nguồn: https://docs.aws.amazon.com/xray/latest/devguide/xray-sdk-daemon-timeline.html
- Ruling Task1.2: dùng aws-xray-sdk-go v1.8.5 (maintenance, security fixes theo AWS) với awsv2.AWSV2Instrumentor. Lambda Active tracing + daemon/context có sẵn, không thêm collector/exporter hoặc hạ tầng OTel trong nền tảng local; cân nhắc ADOT khi có yêu cầu lifecycle instrumentation mới. Test local chỉ xác nhận subsegment và middleware, không khẳng định đã export lên X-Ray AWS.
- Build: hai wrapper gọi Go helper dùng archive/zip SetMode(0755), đảm bảo Linux executable permission trên Windows. Discover cmd thật, không tạo stub nghiệp vụ; chưa có cmd thì fail rõ ràng. Fixture chỉ kiểm compiler/ZIP, không giả lập flow chính.
- Ruling bảo mật: pin ban đầu Go1.26.5 đã cũ dù còn thuộc major được hỗ trợ; scan phát hiện stdlib vulnerabilities, nên kết quả Phase1 dùng minimum/toolchain Go1.26.10, CI cùng patch. Nguồn version chính thức https://go.dev/dl/ và https://go.dev/doc/devel/release; các kết quả 1.26.5 trong Task1.1 là lịch sử, không phải bằng chứng cho bản cuối.
- Bản cuối Task1.1 dùng Go1.26.10 thay pin1.26.5 ban đầu; sửa cùng PR #66 để baseline của Task1.2 cũng yêu cầu bản vá.
- Security review cuối: govulncheck v1.9.0 trên Go1.26.10 trả exit0, 0 symbol/package vulnerabilities; 2 module-only GO-2022-0646 và GO-2022-0635 ở S3 Crypto của SDKv1.55.8 gián tiếp, không import/call bởi ứng dụng. Không suppress cảnh báo; ghi rõ và thêm scan reachable vào CI. Không có IAM/DynamoDB transaction ở Phase1 để review.
- Review độc lập cả hai task không có finding code cần sửa; các yêu cầu export trace lên AWS, handler thật và E2E dành cho phase sau, không suy diễn từ fixture local.


## Final review Task 1.1 — 2026-11-04
- Squash merge PR cha không tự nhận các commit được merge vào nhánh cha sau thời điểm đó. API GitHub xác nhận #65/#66/#67 đều merged, nhưng git tree main c99bd5e chỉ có Phase 0. Phục hồi theo từng task trên các nhánh từ main, không dùng merge-base với lịch sử cũ để đưa lại toàn bộ reset vào PR.
- `netip.ParseAddr` thất bại không có nghĩa host là DNS hợp lệ: IPv4 sai chỉ chứa chữ số/dấu chấm phải bị từ chối. Giữ hỗ trợ canonical IPv4 có trailing dot và DNS có nhãn chữ số; regression test trước sửa RED, sau sửa GREEN.

## Tích hợp Task 1.2 — 2026-11-04
- PR #68 đã đưa Task 1.1 và regression IPv4 lên main. Phục hồi Task 1.2 từ source cũ nhưng giữ nguyên bản sửa domain; các xung đột chỉ nằm trong tài liệu trạng thái và đã hợp nhất lịch sử theo ngày.
- Scan hiện tại có 4 module-only advisories, tăng so với 2 cảnh báo ngày 27/10 do thêm GO-2026-6866/6860 trong x/net; không có symbol/package bị ảnh hưởng. Duy trì bằng chứng hiện tại riêng với nhật ký cũ.

## Static export final review — 2026-11-04
- Next build hiển thị route prerendered không đồng nghĩa artifact static export: scaffold cũ build exit0 nhưng không có out/. Regression tests chạy trên scaffold thật trong thư mục tạm đã xác nhận RED trước sửa.
- Dùng hướng dẫn bundled của next16.4.3 để bật export/trailingSlash/unoptimized. Artifact tests kiểm HTML/404 và các file CSS/JS/images được tham chiếu; không chỉ assert giá trị config. Build trước khi chạy pnpm test.
- CLI local mặc định pnpm12 khác packageManager10.33.2; dùng corepack pnpm@10.33.2 để kiểm nhất quán, CI cài đúng pnpm10.33.2. Không thay dependency/lockfile để né lỗi môi trường.
- #68 và #69 đều qua CI trước khi squash vào main; main hiện có đầy đủ code Phase 1. Static frontend sửa theo PR riêng; không tuyên bố auth/dashboard/Doodle UI đã hoàn tất.


## Phase 2 / Task 2.1 — 2026-11-04
- DynamoDB Local3.3.0 xác nhận PutItem collision và DeleteItem ownership conditions chạy đúng; GSI query ScanIndexForward=false đọc LastEvaluatedKey đến hết. Code được dựng từ PK LINK#, không cần thêm code attribute vào schema lưu trữ. 6 item khoảng220KB buộc pagination thật và kiểm milliseconds ordering.
- API chỉ dùng authorizer.jwt.claims.sub. Body ownerId, bearer chưa xác thực và X-Local-User không tạo danh tính production. List xuất đúng4 fields; missing và khác owner có cùng404. Create nhận một object JSON, chặn null/nonstring/trailing JSON và body>16KiB.
- PutEvents phải kiểm FailedEntryCount và entry riêng, không chỉ HTTP200. Chỉ retry InternalFailure/ThrottlingException (3calls tổng), full jitter50/100ms, context2s và NopRetryer mỗi SDK operation để tránh retry lồng. Transport error có kết quả bất định nên fail closed ngay. Whole handler4s; lỗi EMF không làm publish lại. Nguồn: [AWS result entry](https://docs.aws.amazon.com/eventbridge/latest/APIReference/API_PutEventsResultEntry.html), [SDK timeout/retry](https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/configure-retries-timeouts.html).
- Local harness dùng IP loopback/credentials local cố định, không load operator credential chain. -record-events là fixture giao thức opt-in, chỉ ghi nhận event và ack cục bộ, không tạo analytics hay xác minh delivery AWS. HTTP E2E dùng SDK serializer, real handlers và DynamoDB Local; JWT authorizer/IAM/pipeline vẫn pending.
- Skills ở host hiện tại không còn giống audit lịch sử: chỉ24 thư mục AWS tại ~/.agents/skills; TDD tìm thấy dưới plugin superpowers. Không có planning-with-files/ponytail/differential-review; làm theo AGENTS về trạng thái/review, không tự cài lại.
- SDK v2 modules mới pin theo resolver; core1.47.5/smithy1.28.7 là patch dependency bắt buộc. Scan reachable exit0/0 symbol/package, còn4 module-only warnings đã ghi ở Phase1; không thay dependency ngoài phạm vi để che cảnh báo.


## Phase 2 / Task 2.2 — 2026-11-04
- Terraform1.17.1/AWS provider6.71.0 được xác minh từ host/registry chính thức và pin cùng lockfiles. GSI hash_key/range_key đã deprecated ở provider này; dùng key_schema giữ cùng access pattern GSI1PK/GSI1SK. PK/SK primary vẫn dùng hash_key/range_key hợp lệ. Terraform schema/plan đã kiểm, không đổi nghiệp vụ.
- Plan tests cần mock computed values ở plan phase và ARN/account hợp lệ, không chuyển sang test apply chỉ để assertion đọc được. Bộ cuối8bootstrap+8core PASS, negative tests chặn wildcard trust/environment/state scope và callback/CORS sai.
- API Gateway v2 không được giả định tag inheritance của V1. Child CRUD và tag endpoints không hỗ trợ cùng tag context nên scope bằng API ID/path; first provisioning có ID wildcard trong region, phải pin output ID trong bootstrap trước CD. Tag endpoints hỗ trợ cả ARN URL-encoded và deny thay security tags bằng ForAnyValue:StringEqualsIgnoreCase. Nguồn: [AWS resources/tag inheritance](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-tagging-supported-resources.html).
- Cognito DescribeUserPoolDomain không hỗ trợ resource ARN; tách read wildcard với RequestedRegion. CreateUserPool cần tagging authorization, native signup bị tắt; supported IdP chỉ Google, explicit auth chỉ refresh, không password/SRP. Public code client dùng PKCE do browser triển khai ở Phase5; không có cờ Terraform bắt buộc PKCE riêng. Lite/classic domain version1 tránh thêm paid-tier/branding trong core.
- S3 state policy chỉ Get/Put exact state, Delete chỉ lockfile, ListBucket dev/* để backend liệt kê workspace metadata trong dev/workspaces. Chỉ default state object được cấp; không dùng workspace tạo môi trường mới. Bootstrap local state độc lập được giữ ngoài deployment role. Google secret vẫn ở state/plan dù sensitive; không commit hoặc log.
- Execution roles bị runtime boundary bootstrap giới hạn, deployment không được sửa boundary/trust/bootstrap hay xóa boundary khỏi role. PassRole exact4 role ARNs và PassedToService Lambda; business actions tách theo từng handler. Resolve live alias ignore_changes, private aliases vẫn cập nhật version bình thường.
- IAM Autopilot unavailable: uvx không có, python venv thiếu ensurepip và python pip không có; fallback theo skill qua AWS Service Authorization Reference kiểm104 action names/capabilities, không giả vờ đã chạy Autopilot. Command tái tạo baseline được lưu runbook; không upload IAM.
- Host thiếu make; direct recipe commands PASS, CI không phụ thuộc make. PowerShell7.5 official container parse/preflight invalid endpoint PASS không mạng AWS; AWS Core/OIDC script giữ token trong environment, xác nhận type/issuer/client/expiry, không follow redirect và cleanup chỉ link vừa tạo đúng target ngẫu nhiên. E2E Google/JWT/IAM/STS/X-Ray còn pending.


## Final review Phase 2 — 2026-11-04
- RequestTag không xác lập ownership của Cognito pool: loại wildcard TagResource/CreateUserPool khỏi deployment role. Operator tạo pool lần đầu qua Terraform rồi pin ID vào bootstrap; maintenance chỉ ARN đã pin, deny sửa Project/Environment bất kể hoa/thường.
- AWS provider6.71.0 refresh ZIP Lambda gọi GetFunctionCodeSigningConfig cả khi không cấu hình code signing; bổ sung exact function ARNs.
- Người dùng chốt cursor pagination thay list-all. Một DynamoDB query mỗi request, limit25 mặc định/100 tối đa; cursor owner-bound và validate key shape. Budget2.5MiB inner JSON dự phòng escape của Lambda envelope; test450 URL dài qua real handler kiểm dưới6MiB và không mất/trùng item. GSI eventual consistency, page cuối có thể rỗng khi Limit đúng biên.
- E2E sau delete chỉ retry301, tối đa10lần/250ms;404 kết thúc, status khác fail ngay. Không thêm sleep tùy ý hoặc che500.

- Cognito DescribeIdentityProvider tự bổ sung username=sub và metadata endpoints/method. [Provider6.71.0](https://github.com/hashicorp/terraform-provider-aws/blob/v6.71.0/internal/service/cognitoidp/identity_provider.go) ghi toàn bộ map vào state nên cấu hình chỉ3provider keys bị perpetual diff. Ignore chỉ username và6metadata keys giải quyết actual plan exit2→0, vẫn quản lý credentials/scopes/app mapping.
- IAM Access Analyzer trên policy thực đã pin không có finding; IAM simulation thử nghiệm không cho kết quả dùng được (custom input lỗi, principal result có ARN placeholder), không coi là evidence quyền. Runtime permissions được smoke qua Lambda thực với DynamoDB/EventBridge; GitHub STS/OIDC matrix vẫn pending. Bootstrap local state cần giữ backup tách app backend để tránh deployment role tự quản bootstrap.


## Phase 3 / Task 3.1 — 2026-11-04
- Tách `aggregator.go` trong repository để giữ implementation transaction/reason classifier tập trung; vẫn dùng cùng DynamoDB repository/client/table, không thêm kho dữ liệu hay framework. Mở rộng SDK interface và test doubles ở ba package để bảo toàn unit tests cũ.
- Giữ cố định transaction META/STAT/marker, chỉ ack đúng3reasons và consistent read xác nhận delete. Nil SDK responses fail closed; conflict/throttle/mixed/unknown/thiếu reason phải retry.
- Ngày STAT lấy từ timestamp event chuẩn hóa UTC, TTL từ processing time; event trễ không bị chuyển sang ngày nhận. Envelope invalid source/type/id/code/at bị retry, không tạo item. Dedupe dựa item tồn tại, không dựa thời điểm TTL.
- Local DynamoDB3.3.0 xác nhận transaction/reasons Go SDK, dedupe finite window, sparse marker/STAT, concurrent deliveries và delete race. Marker mất rồi replay đếm lại là giới hạn chủ đích; không thêm generation/outbox.
- Batch có deadline4s; remaining records được retry. Notifier sau commit với budget200ms, không rollback/retry click đã commit nếu notifier hoặc EMF lỗi; production notifier chưa gắn cho tới Phase4.
- HTTP E2E giữ actual publisher entry/detail/EventId và handoff tường minh vào consumer; fixture chỉ thay external event delivery. Không dùng fixture làm evidence AWS EventBridge/SQS/ESM/DLQ hoặc WebSocket.


## Phase3 / Task3.2 — 2026-11-04
- Module event_pipeline sở hữu queue/rule/target/resource policies/redrive allow/ESM và tái dùng module Lambda cho aggregator; root giữ bốn HTTP module targets và merge aggregator vào outputs roles/aliases. Bus lấy đúng name/ARN Task2.2, envelope không transform, không ESM filtering làm ack poison.
- IAM runtime dùng UpdateItem/PutItem/GetItem đúng table và ReceiveMessage/DeleteMessage/GetQueueAttributes đúng ClickQueue; SAR không có action TransactWriteItems. Boundary phải mở UpdateItem/poll queue và role allowlist có thêm aggregator, nếu chỉ cấp inline execution policy sẽ bị boundary chặn.
- Quyền mới khiến tổng inline bootstrap policy11256byte >10240 (regression RED); tách managed policy bootstrap sở hữu, giữ tổng inline<=10240 và managed<=6144 (GREEN). Không cấp deployment role quyền sửa managed policy/bootstrap của chính nó.
- ESM UUID không biết trước; operator provision Terraform lần đầu rồi pin click_event_source_mapping_ids, maintenance exact UUID + lambda:FunctionArn alias live. Không CreateEventSourceMapping/wildcard TagResource để role claim mapping khác. Replacement cần operator/repin; đây là ruling theo nguyên tắc Cognito đã chốt ở Phase2, chưa apply AWS.
- uvx Autopilot vẫn thiếu (exit127), fallback SAR JSON chính thức DynamoDB/Lambda/SQS/EventBridge. Get/Update/Delete ESM hỗ trợ FunctionArn; tag/list tag scope UUID riêng. SSE-SQS được AWS example xác nhận tương thích EventBridge, không nhầm với AWS managed KMS key.
- Task3.2 thêm Scenario Pipeline thật theo ma trận master plan; chỉ parser/preflight offline được chạy. Script đọc config/account trước side effects, HTTP redirect→consistent counter, PutEvents poison→DLQ, đối chiếu event ID và cleanup code/receipt của test; không purge/redrive/tắt mapping. ApproximateReceiveCount ở DLQ được ghi như quan sát, không coi là số lần source consumer nhận.
- Review phát hiện JSON scalar/array ở DLQ làm ContainsKey throw; regression RED→GREEN bằng Read-PipelineEnvelope/NoEnumerate/IDictionary/id string. Container PS7.5 --network none PASS, scoped re-review xác nhận đã sửa. Các ràng buộc quan trọng và giới hạn AWS pending được giữ trong docs/terraform-event-pipeline.md.


## 2026-11-04 — Phase4 / Task4.1
- API Gateway WebSocket dùng events.APIGatewayWebsocketProxyRequest (requestContext.connectionId), không dùng HTTP API v2 envelope. `$disconnect` không cần query code. Schema CONN/META và WSCODE GSI giữ nguyên, TTL tính bằng processing time+7200.
- Pusher đọc count consistent, stream mọi GSI page bằng callback để tránh giữ toàn bộ subscribers trong RAM. Query projection PK không bỏ giới hạn1MB evaluated items; test dùng6item*220KB buộc pagination thực tế. GSI vẫn eventual consistency.
- AWS SDK v2 Management API v1.41.3 giữ nguyên dependency graph hiện có; BaseEndpoint phải là HTTP(S) management URL có stage, SDK nối @connections và ký SigV4. Runtime test và E2E fixture đã kiểm path, auth header, payload và typed GoneException từ HTTP410.
- Giữ budget notifier200ms có sẵn: broadcast lớn/cold request có thể không hoàn tất, không có durable delivery. Get/Query/Post/cleanup failures đều log và cô lập sau commit; chỉ410xóa connection. Errors không làm đếm lại click; duplicate/deleted không push.
- ASL dùng trực tiếp ModerateCheckArn nên kết quả không có Payload envelope. CheckSafety trả code/target/safe và không fetch URL; substring malware/phishing/evil giữ nguyên case-insensitive như nguồn. Validation input thiếu trả lỗi để Catch/Fail, không ngầm duyệt link.
- ASL updateItem không condition theo đặc tả nên có thể upsert META không hợp lệ; runbook bắt buộc tạo/kiểm META trước, không xóa song song. Retry3lần nghĩa là tối đa4invocations, chờ1/2/4giây; execution history thật chờ Task4.2 và AWS authorization.
- Host hiện không có các workflow skills planning-with-files/ponytail/TDD/verification-before-completion; không coi audit cũ là bằng chứng hiện diện. Dùng aws-serverless và quy trình AGENTS/test-first trực tiếp.


## 2026-11-04 — Phase4 / Task4.2
- Root giữ4HTTP Lambdas, thêm3realtime functions, aggregator vẫn event_pipeline. Tách websocket_endpoint wss cho client và management_endpoint https cho SDK; không có dependency cycle vì API không phụ thuộc aggregator. WebSocket native proxy bỏ HTTP2.0 payload, invoke permission pin prod/route/live.
- State machine STANDARD render cùng ASL; role trust xây ARN từ tên/account/region để không tham chiếu vòng. Boundary riêng chỉInvoke moderate-check:live/UpdateItem table, Lambda không đượcInvoke hoặcStartExecution.
- Bootstrap mở ManageConnections chỉ khi websocket_api_ids cóIDexact; unknown API không thành wildcardpush. API child/tag permissions hợp nhất pinned HTTP/WS IDs. Operator tạo tài nguyên lần đầu, pinIDs/applybootstrap rồi mới CD/push.
- Regression fully-pinned inline12314>10240 tái hiện thực; chuyển AppFunctions/API grants sang operator-owned managed policies, giữ policy behavior/tag protections. Test kiểm inlineaggregate và từngmanaged/boundary quota, đủ8Lambda allowlist.
- Đọc provider6.71.0 source phát hiện ValidateStateMachineDefinition khi plan và ListStateMachineVersions khi refresh; thêm regression RED→GREEN. SAR xác minh Validate không có resource scope, statement* chỉRequestedRegion; ListVersions chỉexactmachine. Không coi mock-provider plan là AWS permissions proof.
- Autopilot uvx/pip unavailable; dùng SAR fallback và source provider chính thức. ScenarioRealtime validate config/missingtoken offline, thật sẽ kiểm STS/GetApi owner trước khi gửi bearer, đợi GSI/subscriber và execution history; không sửa hạ tầng qua CLI.
- Runtime logging Lambda/X-Ray giữ module chung; WebSocket account-level logging và SFN CloudWatch execution logging/trace đểPhase6, tránh cấp thêm service-role permissions ngoài contract hai nghiệp vụ ởPhase4.


## 2026-11-04 — Sửa lost-response cleanup Task4.2
- StartExecution không truyền name dùng UUID do AWS tạo nên response bị mất thì không biết execution ARN. Tạo tên e2e-code-GUID trước request cho phép suy ra ARN, đăng ký trước khi gửi và giữ CLI retries idempotent theo name/input của STANDARD.
- Không xem lỗi transport là bằng chứng request bị từ chối. Cleanup Describe trước; chỉ stop RUNNING, rồi Describe để xác nhận terminal. Stop bị mất response vẫn có thể đối soát ABORTED. Execution chưa tồn tại/không đọc được/still RUNNING/PENDING_REDRIVE đều giữ META và in ARN, không suy diễn an toàn để xóa.
- Regression tests chạy hàm Start-RealtimeModeration và finally lấy trực tiếp từ AST nguồn để không khởi động luồng AWS/HTTP thật; mock tại dependency boundary. Tám trường hợp bao gồm start mất response, workflow đã xong, không rõ trạng thái, stop bị từ chối/mất response, poll lỗi, trạng thái chưa kết thúc và success.

## Phase 5 / Task 5.1 — 2026-11-05

- Task 5.1 dùng native Next.js App Router + Tailwind/CSS hiện có, không thêm component library. Static export tiếp tục dùng `output: "export"`, `trailingSlash: true`, `images.unoptimized: true`.
- Browser auth chỉ dùng Cognito authorization code + PKCE S256 với Google IdP; không có password/signup flow và không đưa secret vào public config.
- Builder review đã chốt thêm: đối chiếu `id_token.nonce`, lưu expiry và dọn query callback bằng `history.replaceState`; service URL production chỉ HTTPS/WSS, origin-only và reject placeholder/query/fragment.


## Phase 5 / Task 5.3 — Recreated branch — 2026-11-05

- Cách tránh conflict: dựng branch mới từ `origin/main`, áp dụng implementation hosting theo file, không mang merge commit `86a40ad` hoặc các thay đổi trạng thái đã xung đột của PR #80.
- Module giữ S3 private + CloudFront OAC SigV4, bucket policy giới hạn distribution ARN, cache policy tách HTML/API assets và CloudFront Function rewrite cho static routes.
- Bộ kiểm Terraform mock xác nhận frontend hosting wiring cùng core/event/realtime contracts vẫn pass 19/19; chưa apply AWS trong bước tạo PR.

## Phase 5 / Task 5.4 — Shrnk frontend refresh — 2026-11-05

- Frontend dùng một `LinkIcon` chung cho branding và Navbar; trạng thái đăng nhập điều khiển các liên kết Dashboard, đăng xuất và landing create form.
- Dashboard giữ workspace analytics links, bỏ trải nghiệm My Links/create form trùng lặp; route `/architecture/` được export tĩnh.
- PR #82 giữ static export và API/WebSocket thật; không thêm password/signup flow hoặc runtime server-side.


## 2026-11-05 — Google login local và page metadata

- Checkout chạy dev vẫn ở `4e7c590`, trong khi bản tách auth config đã merge qua PR #84 (`004b935`). Tạo branch fix mới từ `origin/main`, giữ các diff backend/Terraform có sẵn ngoài commit frontend.
- Dev server chưa có `NEXT_PUBLIC_*`; bổ sung sáu giá trị public vào `.env.local` từ Terraform outputs và callback/logout localhost đã được Cognito allowlist. File local không được commit; thêm `.env.example` và hướng dẫn cấu hình để tái lập.
- WebSocket URL thật có `/prod`, parser cũ chỉ nhận origin. Test hồi quy RED với lỗi `NEXT_PUBLIC_WS_URL must be an origin URL without a path`, GREEN sau khi giữ pathname cho WebSocket; vẫn từ chối credentials/query/fragment và WS không TLS ngoài loopback.
- Đọc Google JSON và Cognito IdP trong bộ nhớ: client ID và secret đều khớp, không cần thay đổi tài nguyên AWS. Không in secret, không đưa JSON vào repository/assets.
- Logo và favicon SVG dùng chung `/icon.svg`; ICO dự phòng được rasterize từ cùng SVG. Metadata static có title template `%s | Shrnk`, trang chủ giữ tagline riêng.


## 2026-11-05 — Publish website sau PR #85

- Website người dùng yêu cầu cập nhật là CloudFront `d2aamyg2mt28wl.cloudfront.net` hiện hữu, tài nguyên mang tên `dev`; không tạo environment mới hoặc thay backend.
- Publisher và hosting smoke còn giữ route `/links/` đã được bỏ ở frontend refresh. Đổi preflight/smoke sang `/architecture/`; `bash -n`, publisher thật và hosting smoke PASS.
- Publisher export env ưu tiên hơn `.env.local`: bundle production đã kiểm callback/logout CloudFront, WebSocket `/prod`, không có callback localhost hoặc Google client secret.
- Bản mutable objects trước publish được sao lưu ở `/tmp/shrnk-frontend-rollback-20261105/`, hashed assets cũ được giữ trên S3. Không chạy Terraform apply hoặc load test.

## 2026-11-05 — Nguyên nhân OAuth callback state mismatch

- `getAccessToken()` dùng `clearAuthSession()` khi token chưa có/hết hạn, xóa cả PKCE đang chờ. Navbar được mount trước AuthCallback, nên callback mất state/verifier dù URL chứa state đúng. Tái hiện cả bằng unit test RED và browser production với state đúng của phiên chẩn đoán.
- Sửa đọc token chỉ dọn hai khóa access-token/expiry; logout và lỗi callback vẫn dọn cả PKCE. Không bỏ kiểm tra state/nonce hay thay storage sang localStorage.
- Component integration bằng React DOM + jsdom phát hiện thêm StrictMode effect replay: effect thứ hai đọc query đã bị redaction rồi xóa phiên, trong khi request đầu đang chờ. Giữ một promise hoàn tất bằng `useRef`, chỉ đọc query/redact/exchange một lần; effect đang active nhận kết quả.
- Thêm jsdom ở devDependencies; helper test transpile TSX bằng TypeScript sẵn có và nạp Next components thật. Chỉ mock fetch tại ranh giới token endpoint, không mock Navbar/AuthCallback/auth functions. Tests kiểm code/verifier/redirect, delayed response, redirect dashboard đúng một lần, dọn PKCE sau thành công, state/nonce sai bị từ chối.


## 2026-11-05 — UI refinement, project review và production rollout

- **Status: complete (implementation/deployed)**. Yêu cầu: đăng nhập về trang chủ; sửa dashboard nhấp nháy; layout chung; menu active/hover; login tối giản; font Lobster; review toàn project, sửa lỗi và triển khai production.
- Nhánh `fix/phase-05-ui-review-production` tạo từ `origin/main` (`2253a3c`), kế thừa hai commit sửa PKCE #87 để giữ luồng đăng nhập an toàn. Giữ nguyên các diff có sẵn để rà soát, không tự hoàn nguyên.

- Nguyên nhân nhấp nháy: mọi socket `onopen` gọi reload, `loading=true` unmount bảng, đóng socket rồi remount/kết nối lại. Background reload nay giữ bảng, chia sẻ request đang chạy; `ClickCell` giữ max giữa counter HTTP và push. Regression RED trên code cũ → GREEN, có reconnect thật bằng timer và HTTP lỗi.
- Race sau xóa: snapshot đã bắt đầu trước DELETE có thể khôi phục row sau khi DELETE thành công. Giữ tập code đã xóa trong vòng đời workspace và lọc snapshot; regression RED → GREEN.
- Form landing từng bỏ kết quả qua `onCreated` no-op. Kết quả API thật nay hiển thị short URL và Copy, có component integration ở ranh giới fetch/clipboard.
- Go build ban đầu fail vì dòng `*dest = strings.TrimRight(normalized, "/")` trong config bị xóa ở diff có sẵn. Khôi phục đúng dòng từ baseline; unit config có sẵn xác nhận hai endpoint được gán và bỏ trailing slash. Các diff dọn chú thích backend/Terraform/PowerShell và xóa tài liệu có sẵn được giữ ngoài PR; không tự hoàn nguyên.
- Terraform test tại root có tfvars AWS thật làm 3 assertion về unpinned API sai fixture. Chạy bản sao sạch không tfvars/backend/state, kèm ASL và ZIP thật: 19 bootstrap + 19 dev PASS. Actual plan chỉ khác filename/hash Lambda do rebuild, không có thay đổi cấu hình/replace/destroy; không apply vì backend behavior không đổi và smoke deployed đã PASS.
- Lobster TTF đầy đủ hỗ trợ tiếng Việt từ Google Fonts, kèm OFL; `next/font/local` preload/self-host, không gọi dịch vụ font ngoài runtime.

## 2026-11-05 — Conflict PR #87
- PR #86 và #87 chèn nội dung tại cùng vị trí trong bốn file trạng thái. Conflict chỉ ở tài liệu; publisher/hosting routes từ main được giữ. Không thay state/nonce/PKCE hoặc callback effect handling.


## 2026-11-05 — Conflict PR #88
- Đồng bộ qua merge head #87 đã chứa main #86 để giữ cùng lịch sử tổ tiên. Conflict ở bốn file trạng thái và vùng preflight publisher; giữ kiểm thử trước upload cùng route architecture. Mã `frontend/` không đổi so với PR #88 đã deploy.
