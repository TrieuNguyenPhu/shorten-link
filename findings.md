# Ghi chú hiện tại

- Task 5.1 dùng native Next.js App Router + Tailwind/CSS hiện có, không thêm component library. Static export tiếp tục dùng `output: "export"`, `trailingSlash: true`, `images.unoptimized: true`.
- Browser auth chỉ dùng Cognito authorization code + PKCE S256 với Google IdP; không có password/signup flow và không đưa secret vào public config.
- Builder review đã chốt thêm: đối chiếu `id_token.nonce`, lưu expiry và dọn query callback bằng `history.replaceState`; service URL production chỉ HTTPS/WSS, origin-only và reject placeholder/query/fragment.
- Backend và Terraform lõi đã có kiểm thử local. Production rollout vẫn phụ thuộc frontend hosting, dashboard, observability, CI/CD và xác minh AWS E2E.
- AWS secret hygiene: giá trị Google OAuth client secret đã xuất hiện trong một tool output trong phiên. Người vận hành cần xoay secret đó ở Google Cloud và cập nhật Cognito trước khi dùng tiếp.

- Landing auth dùng sessionStorage token để hiện `HomeCreateLink`; Navbar signed-in hiển thị Dashboard/Đăng xuất và không hiện nút đăng nhập.
- Dashboard dùng `LinksWorkspace` ở chế độ danh sách, không còn `CreateLinkForm`; route `/architecture/` là static page.
