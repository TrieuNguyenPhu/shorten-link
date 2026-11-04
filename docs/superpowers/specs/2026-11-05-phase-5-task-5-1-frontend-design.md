# Phase 5.1 Frontend Authentication and Pages Design

## Intent

Hoàn thiện ba trang frontend đầu tiên của hệ thống URL shortener: landing, đăng nhập Google qua Cognito và callback OAuth. Các trang phải xuất tĩnh bằng Next.js App Router để phục vụ từ S3 private + CloudFront, đồng thời làm rõ trạng thái loading, lỗi và hướng dẫn tiếp theo cho người dùng.

## Goals

- Giữ Next.js `output: "export"`, `trailingSlash: true`, ảnh unoptimized và không yêu cầu Node.js runtime khi phục vụ.
- Chỉ cung cấp nút **Tiếp tục với Google** qua Cognito `/oauth2/authorize` với authorization code + PKCE S256, `state`, `nonce` và scope `openid email profile`.
- Xác thực callback bằng `state`/PKCE transaction trong `sessionStorage`, đổi code lấy Cognito access token qua `/oauth2/token`, dọn vật liệu tạm và xử lý lỗi/lặp callback an toàn.
- Xây landing, login và callback theo phong cách Hand-Drawn / Doodle & Sketch UI, responsive, keyboard accessible, tương phản rõ và tôn trọng `prefers-reduced-motion`.
- Giữ public build config không chứa secret và từ chối endpoint/client ID placeholder.

## Non-goals

- Chưa xây dashboard, My Links, gọi HTTP API thật hay WebSocket realtime; các phần đó thuộc Task 5.2.
- Chưa tạo S3 bucket, CloudFront OAC, cache policy hoặc rewrite; thuộc Task 5.3.
- Không thêm password login, signup riêng, Google token forwarding hoặc Next.js server runtime.

## Constraints

- Frontend dùng TypeScript, App Router, Tailwind CSS và dependency hiện có trong `frontend/package.json`.
- Cognito app client là public client không có secret; browser chỉ lưu access token ngắn hạn trong session storage.
- Redirect callback build-time lấy từ `NEXT_PUBLIC_AUTH_REDIRECT_URI`; production URL sẽ do Terraform output cung cấp.
- Callback đối chiếu `nonce` trong Cognito `id_token` với transaction trước khi lưu access token; ID token chỉ dùng cho kiểm tra này và không gửi tới API.
- Access token được lưu kèm thời điểm hết hạn (`expires_in`); token hết hạn bị coi là không hợp lệ và bị dọn khỏi session.
- Callback đọc query đúng một lần, dùng `history.replaceState` để xóa `code`, `state`, `error` và `error_description` khỏi URL trước khi hiển thị kết quả.
- Public env được tham chiếu tĩnh từ `process.env.NEXT_PUBLIC_*` để Next inline vào static bundle; URL API/WS/Cognito phải là origin HTTPS/local hợp lệ, không chứa placeholder hoặc query/fragment.
- Không dùng `innerHTML` cho dữ liệu OAuth hay URL; React escaping và URL validation phải được giữ nguyên.

## Approaches considered

1. **Native Next.js + CSS/Tailwind hiện có (chọn):** dùng component App Router nhỏ, CSS nét vẽ tay tự định nghĩa và auth helper thuần browser. Cách này khớp static hosting, không thêm dependency và giữ bundle nhỏ.
2. **Component library:** giúp dựng UI nhanh hơn nhưng thêm dependency/bundle và không đem lại lợi ích cần thiết cho ba trang có bố cục đơn giản.
3. **Chỉ làm auth rồi hoãn UI:** giảm phạm vi thay đổi nhưng không đáp ứng yêu cầu Task 5.1 về landing/login/callback và trạng thái dễ đọc.

## Design decisions

- `src/lib/config.ts` là cổng duy nhất đọc public environment và kiểm tra HTTPS ngoài local.
- `src/lib/auth.ts` chứa các hàm thuần có dependency rõ ràng trên `StorageLike`, để unit test state mismatch, thiếu/lặp code, token exchange lỗi và logout mà không cần browser thật.
- `src/app/login/page.tsx` khởi tạo URL Google khi người dùng bấm nút; không tự chuyển hướng khi render.
- `src/app/auth/callback/page.tsx` chỉ chạy client-side, hiển thị loading/error, đổi code một lần và chuyển về `/dashboard/`. Task 5.1 tạo route `/dashboard/` placeholder tối thiểu để callback không rơi vào 404; Task 5.2 sẽ thay bằng dashboard thật.
- CSS dùng nền giấy sáng, border/dashed accent có kiểm soát, focus ring rõ, animation chỉ bật khi người dùng không yêu cầu reduced motion.

## Verification

- Unit test Node cho PKCE transaction, callback validation, token exchange, config validation và logout cleanup.
- `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build` và kiểm tra `out/` có HTML/static assets độc lập.
- Không chạy `terraform apply`, AWS E2E hay load test trong Task 5.1.
