# Shrnk Frontend Updates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Cập nhật branding, navigation, CTA, dashboard và landing auth flow theo yêu cầu Shrnk.

**Architecture:** Giữ Next.js App Router static export. Auth visibility đọc từ sessionStorage trong Navbar và landing form; dashboard dùng workspace hiện có với prop tắt form.

**Tech Stack:** Next.js, React, TypeScript, Tailwind CSS, Node test runner.

**Spec:** `docs/superpowers/specs/2026-11-05-shrnk-frontend-updates-design.md`

## Global Constraints
- Giữ `output: "export"` và không dùng runtime server.
- Giữ Google-only Cognito PKCE.
- Không thay đổi API/backend.
- Copy giao diện tiếng Việt; tên sản phẩm `Shrnk`.

## Review Focus
- Signed-in Navbar phải không hiển thị nút đăng nhập.
- Landing phải không gọi API khi chưa có token.
- Dashboard không render form tạo link.
- Static export phải sinh route `/architecture/`.
- Không còn mũi tên trong button text hoặc chú thích nhỏ dưới đối tượng.

### Task 1: Lock requested copy and routes with tests
**Files:** Modify `frontend/tests/pages.test.mjs`, `frontend/tests/static-export.test.mjs`.
- [ ] Thêm assertion cho Shrnk, Source Code, Kiến trúc dự án, architecture route, không có My Links/arrow button copy.
- [ ] Chạy test để xác nhận RED.

### Task 2: Shared branding and auth-aware navigation
**Files:** Modify `frontend/src/components/Navbar.tsx`; create `frontend/src/components/LinkIcon.tsx` if needed.
- [ ] Dùng icon link cho logo/banner.
- [ ] Đọc `getAccessToken` trong client effect và hiển thị Dashboard/Đăng xuất khi signed-in.
- [ ] Cập nhật title và copy.

### Task 3: Landing and architecture pages
**Files:** Modify `frontend/src/app/page.tsx`; create `frontend/src/app/architecture/page.tsx`; create `frontend/src/components/HomeCreateLink.tsx`.
- [ ] Đổi CTA và xóa mũi tên/chú thích nhỏ.
- [ ] Thêm form tạo link dưới giới thiệu khi signed-in.
- [ ] Thêm route kiến trúc dự án.

### Task 4: Dashboard and residual copy cleanup
**Files:** Modify `frontend/src/features/links/LinksWorkspace.tsx`, `frontend/src/features/links/CreateLinkForm.tsx`, `frontend/src/features/links/LinksTable.tsx`, `frontend/src/app/layout.tsx`, login page.
- [ ] Tắt form ở dashboard.
- [ ] Đổi nhãn My Links thành Dashboard/Danh sách link.
- [ ] Xóa mũi tên và các dòng chú thích nhỏ.

### Task 5: Verify
- [ ] Chạy `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build` trong `frontend/`.
- [ ] Cập nhật `PROJECT_STATUS.md`, `task_plan.md`, `progress.md`, `findings.md`.
