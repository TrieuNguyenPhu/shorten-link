# Phase 6.1 — Giám sát và CodeDeploy canary

Hai module mới được nối vào root dev, bootstrap cấp boundary riêng cho role CodeDeploy và managed policy triển khai. Chưa apply hoặc kiểm chứng dịch vụ AWS trong task này.

## Giám sát

- Giữ alarm Errors và Throttles của resolve: Sum ≥ 1, period 60s, một evaluation period, missing=notBreaching. Dimension FunctionName bao gồm mọi version; lỗi invoke trực tiếp version cũng có thể dừng rollout.
- Thêm alarm ClickDLQ visible ≥ 1 (Maximum), ClickPublishFailed ≥ 1 (Sum). Dashboard có Invocations/Errors/Throttles, Duration p50/p99, backlog/oldest-age/DLQ, LinkResolved và ClickDiscardedDeleted.
- EMF đúng namespace `UrlShortener` và dimension `service=resolve-link` hoặc `click-aggregator`. LinkResolved là publish được chấp nhận, không phải số click đã commit hay unique users. Deleted-discard là tín hiệu đối soát, không mặc định là sự cố.
- X-Ray Active, quyền write traces và retention Logs mặc định 14 ngày tiếp tục dùng cấu hình Lambda hiện có. Không tạo sampling rule hoặc thay middleware.
- SNS topic mặc định không subscription. Chỉ đặt `alarm_email_addresses` khi chủ địa chỉ đồng ý nhận; SNS gửi yêu cầu xác nhận khi apply. Topic trống không có người nhận. Chưa tạo subscription hay gửi email trong phiên này.
- SLO p99/availability chỉ là mục tiêu; cần traffic đại diện và số liệu AWS trước khi công bố đạt SLO. Dashboard, alarms, Logs, traces và SNS có chi phí; kiểm retention và ngân sách trước bật production.

## Quyền và thứ tự triển khai

1. Operator review/apply bootstrap riêng khi được phép. Boundary `${name_prefix}-canary-boundary` cho phép CodeDeploy chỉ GetAlias/UpdateAlias/GetProvisionedConcurrencyConfig trên resolve `:live` và DescribeAlarms (API cần wildcard để đọc composite alarms).
2. App root dùng boundary ARN xác định theo account/prefix, có thể truyền output `canary_boundary_arn` tường minh. Role app deployment không tự sửa bootstrap. Trust CodeDeploy chỉ đúng SourceAccount và ARN deployment group; ARN suy ra từ Lambda account/region và prefix, không tạo dependency cycle.
3. Operator review saved plan của dev; apply khi được phép. Không chuyển hay tái tạo alias: alias resolve do module Lambda quản lý sẵn, ignore `function_version`/`routing_config`. HTTP API integration và permission vẫn qualified `live`.
4. CodeDeploy application Lambda và deployment group dùng `CodeDeployDefault.LambdaCanary10Percent5Minutes`, error + throttle alarms; dừng khi không đọc được alarm, rollback khi failure/alarm/request stop.
5. Pipeline Task 6.2 lấy `CurrentVersion` từ alias thật, `TargetVersion` từ `resolve_published_version`, same-version no-op. Một writer duy nhất, kiểm active deployments trước apply và trước rollout. Chờ success hoặc rollback terminal; khi lỗi không reset alias thủ công, không publish frontend.

Outputs mới: `alarm_topic_arn`, `canary_alarm_names`, `dashboard_name`, `codedeploy_application_name`, `codedeploy_deployment_group_name`, `resolve_function_name`; bootstrap thêm `canary_boundary_arn`.

## Kiểm chứng

Terraform mock-plan tests kiểm ngưỡng/dimension, không đăng ký email mặc định, widget, config canary/rollback, boundary/PassRole và quota IAM. Chúng không chứng minh alarm phát tín hiệu hoặc rollback thật. Task 6.2 cung cấp script và runbook cho rollout thành công, cùng fault-induced rollback trong sandbox được phép; giữ trạng thái cloud pending đến khi có deployment ID, alarm history và alias evidence.

Nguồn: [template gốc](https://github.com/nghiadaulau/serverless-url-shortener-aws/blob/main/template.yaml), [CodeDeploy Lambda configs](https://docs.aws.amazon.com/codedeploy/latest/userguide/deployment-configurations.html), [alarm stop](https://docs.aws.amazon.com/codedeploy/latest/userguide/monitoring-create-alarms.html), [AppSpec](https://docs.aws.amazon.com/codedeploy/latest/userguide/reference-appspec-file-structure-resources.html), [service role Lambda](https://docs.aws.amazon.com/aws-managed-policy/latest/reference/AWSCodeDeployRoleForLambdaLimited.html), [Terraform provider 6.71.0](https://github.com/hashicorp/terraform-provider-aws/blob/v6.71.0/internal/service/deploy/deployment_group.go).

Nguồn bổ sung: [CodeDeploy confused deputy prevention](https://docs.aws.amazon.com/codedeploy/latest/userguide/security_confused_deputy.html). Test hồi quy đã xác nhận RED khi thiếu SourceAccount/SourceArn và GREEN sau khi pin exact account/group.
