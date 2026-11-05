# Tiến độ

2026-11-05: Task 5.1 hoàn tất ở mức implementation/local. Frontend có landing, login Google-only, callback state/nonce/PKCE/expiry và dashboard handoff; build static tạo đủ `/`, `/login/`, `/auth/callback/`, `/dashboard/`, `404`. Cổng `pnpm lint`, `pnpm typecheck`, `pnpm test` (12 pass), `pnpm build` và artifact tests PASS. Không chạy Terraform apply, hosting CloudFront hoặc Google browser federation E2E.

2026-11-05: Kiểm tra full Go suite/vet sau Task 5.1 bị fail trước khi compile các package do diff có sẵn ở `backend/internal/config/config.go:27` (`dest` khai báo nhưng không dùng sau khi xóa phép gán). Giữ lỗi ngoài phạm vi, chưa sửa hoặc commit chung với frontend.

2026-11-05: Release checkout sạch dùng để build/deploy chạy `go test -race ./...` và `go vet ./...` PASS. Working tree gốc vẫn giữ các backend edits chưa commit để không trộn phạm vi.

2026-11-05: Hoàn thiện Task 5.2 ở commit `babf5af`. Frontend API client xử lý 400/401/404/429/5xx, cursor đến hết, session expiry, dashboard/My Links thật và hook WebSocket reconnect có snapshot reload, max merge, dọn socket. `pnpm lint`, `pnpm typecheck`, `pnpm test` (17 pass), static build và signed-out Playwright E2E pass.

2026-11-05: Hoàn thiện Task 5.3 từ commit `ea53ccd` với publisher pin fix `0c624b0`. Terraform dev plan/apply tạo S3 private + CloudFront OAC/rewrite/cache, tự thêm callback/logout/CORS CloudFront vào Cognito/API. Bootstrap pin realtime IDs và boundary đã apply. Publisher dùng pnpm 10.33.2, upload hashed assets trước HTML và invalidate/wait. Hosting smoke và browser smoke AWS pass.

2026-11-05: AWS realtime smoke dùng link test đã tạo, nhận HTTP 301, EventBridge → SQS → aggregator và WebSocket `{code,clicks:1}`, sau đó delete owner và bounded 404. Distribution `d2aamyg2mt28wl.cloudfront.net` trạng thái Deployed. Google browser federation thật và access-token E2E còn pending vì không có test identity/token trong môi trường.

2026-11-05: Bắt đầu Phase 5 trên nhánh `feat/phase-05-task-01-nextjs-auth-and-pages`; chốt phạm vi Task 5.1 cho static landing/login/callback, Google-only Cognito PKCE và Doodle UI. Đã ghi thiết kế tại `docs/superpowers/specs/2026-11-05-phase-5-task-5-1-frontend-design.md`; chưa chạy AWS hoặc Terraform apply.

2026-11-04: Rà soát toàn repo, giữ tài liệu giới thiệu/vận hành, loại bỏ tài liệu lịch sử dài và dọn comment dòng trong mã nguồn.

2026-11-05: Cập nhật frontend theo yêu cầu Shrnk: logo/icon link dùng chung, Navbar nhận biết phiên đăng nhập, Source Code trỏ GitHub, route Kiến trúc dự án, landing tạo link khi signed-in, dashboard chỉ hiển thị danh sách, bỏ My Links và mũi tên/chú thích nhỏ. `pnpm lint`, `pnpm typecheck`, `pnpm build`, `pnpm test` (16 pass).
