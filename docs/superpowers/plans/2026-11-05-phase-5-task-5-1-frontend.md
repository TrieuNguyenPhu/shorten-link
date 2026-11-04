# Phase 5.1 Frontend Authentication and Pages Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Hoàn thiện frontend Next.js static export với landing, Google-only Cognito PKCE login và OAuth callback theo Hand-Drawn / Doodle & Sketch UI.

**Architecture:** Các trang App Router được prerender thành HTML/CSS/JS bằng `output: "export"`; phần cần browser API nằm trong Client Components nhỏ. Auth helper thuần browser dùng `StorageLike` để kiểm thử state/nonce/PKCE, đổi code lấy Cognito access token và logout mà không đưa secret vào frontend.

**Tech Stack:** Next.js 16.4.3, React 19.3.0, TypeScript, Tailwind CSS 4, Node test runner, pnpm 10.33.2.

**Spec:** `docs/superpowers/specs/2026-11-05-phase-5-task-5-1-frontend-design.md`, `docs/architecture-and-contracts.md` mục 7.

## Global Constraints

- Next.js phải dùng `output: "export"`, `trailingSlash: true`, `images.unoptimized: true`.
- Chỉ Google qua Cognito `/oauth2/authorize`, authorization code + PKCE S256, scope `openid email profile`.
- Không có password login, signup riêng, Server Actions, API Routes, SSR hoặc secret trong `NEXT_PUBLIC_*`.
- Redirect callback production có dạng `/auth/callback/`; mọi URL public phải HTTPS ngoài local.
- UI phải responsive, keyboard accessible, tương phản rõ và tôn trọng `prefers-reduced-motion`.

## Review Focus

- Callback thiếu hoặc lặp `code`, sai/mất `state`, PKCE transaction hết hạn: hiển thị lỗi rõ và không lưu token.
- Token endpoint trả non-2xx, JSON thiếu `access_token`, thiếu/sai `id_token.nonce` hoặc redirect/logout URL sai: không làm hỏng trang tĩnh và không nuốt lỗi.
- Access token hết hạn theo `expires_in`: `getAccessToken` trả `null` và dọn session, buộc người dùng đăng nhập lại.
- Cấu hình build thiếu biến, chứa placeholder/client ID giả hoặc URL có query/fragment sai hợp đồng: build/test fail sớm, không tạo artifact có endpoint giả.
- Browser API chỉ được gọi sau hydration trong Client Component; static build phải chạy được khi `window` không tồn tại.
- Reduced motion, focus keyboard và mobile layout vẫn giữ thứ bậc nội dung và trạng thái đọc được.

### Task 1: Public Configuration and PKCE Contract

**Files:**
- Modify: `frontend/src/lib/config.ts`
- Modify: `frontend/src/lib/auth.ts`
- Create: `frontend/tests/config-auth.test.mjs`

**Interfaces:**
- Consumes: `PublicConfig`, `StorageLike`, `PkceTransaction` hiện có.
- Produces: `getPublicConfig(environment)`, `buildGoogleAuthorizationUrl(config, storage)`, `validateCallback(storage, params)`, `exchangeCodeForToken(config, code, storage)`, `logout(config, storage)` với lỗi rõ ràng.

- [x] **Step 1: Write the failing tests**
  - Kiểm HTTPS/local allowlist, bắt buộc đủ sáu biến public, từ chối endpoint/domain chứa `placeholder`/`example`, chuẩn hóa redirect/logout trailing slash, yêu cầu callback pathname `/auth/callback/` và origin-only cho API/WS/Cognito.
  - Kiểm authorization URL có `identity_provider=Google`, `response_type=code`, `scope=openid email profile`, `code_challenge_method=S256`, `state`, `nonce`, `code_challenge` và lưu transaction.
  - Kiểm callback thiếu code/sai state/thiếu transaction bị từ chối.
  - Kiểm token exchange non-2xx/thiếu access token/thiếu hoặc sai `id_token.nonce` bị từ chối; thành công lưu access token, expiry và xóa PKCE transaction.
  - Kiểm `getAccessToken` trả `null` và dọn session khi expiry đã qua.

- [x] **Step 2: Run tests to verify they fail**

Run: `pnpm --dir frontend test`

Expected: FAIL ở các assertion về config validation, URL/error contract hoặc token exchange chưa được triển khai đầy đủ.

- [x] **Step 3: Implement the minimal contract**
  - Giữ dependency injection qua `StorageLike` và `fetch` global; không gọi `window`, `sessionStorage` hoặc `crypto` trong lúc module được import ở build server.
  - Parse URL bằng `URL`, chỉ cho HTTPS ngoài localhost/127.0.0.1, origin-only cho service URLs, callback path `/auth/callback/` và trả lỗi có tên biến.
  - Decode `id_token` payload chỉ để so khớp `nonce`; dọn PKCE transaction sau exchange thành công hoặc callback lỗi cuối cùng; không lưu/forward Google ID token.
  - Dùng tham chiếu tĩnh `process.env.NEXT_PUBLIC_*` khi tạo environment mặc định để Next export inline đúng giá trị.

- [x] **Step 4: Run tests to verify they pass**

Run: `pnpm --dir frontend test`

Expected: PASS toàn bộ test auth/config hiện có.

### Task 2: Shared Layout and Doodle Visual System

**Files:**
- Create: `frontend/src/components/Navbar.tsx`
- Modify: `frontend/src/app/layout.tsx`
- Modify: `frontend/src/app/globals.css`
- Modify: `frontend/src/app/page.tsx`

**Interfaces:**
- Consumes: `getPublicConfig`, `startGoogleLogin`, `logout`, static `Link` routes.
- Produces: shared shell with accessible brand/nav, `PaperCard`, `SketchButton` styling conventions and landing route `/`.

- [x] **Step 1: Write the failing static contract tests**
  - Assert source/pages contain exactly one Google login label, no password/signup form or `innerHTML`, and include accessible labels/focusable links.
  - Assert generated home HTML includes landing heading, `/login/` link and no Next starter copy.

- [x] **Step 2: Run the focused test to verify it fails**

Run: `pnpm --dir frontend test`

Expected: FAIL because the starter page and layout do not contain the product copy or accessibility contract.

- [x] **Step 3: Implement shared shell and landing page**
  - Keep layout as Server Component; place login/logout event handling in a small Client Component.
  - Use a warm paper background, ink borders, hand-written accent labels, semantic landmarks, visible focus ring and reduced-motion media query.
  - Landing copy explains short links and realtime click visibility without mock metrics or fake links.

- [x] **Step 4: Run lint and static tests**

Run: `pnpm --dir frontend lint && pnpm --dir frontend test`

Expected: PASS with no accessibility/lint errors.

### Task 3: Login and OAuth Callback Pages

**Files:**
- Create: `frontend/src/app/login/page.tsx`
- Create: `frontend/src/app/auth/callback/page.tsx`
- Create: `frontend/src/app/dashboard/page.tsx`
- Create: `frontend/src/components/GoogleLoginButton.tsx`
- Create: `frontend/src/components/AuthCallback.tsx`
- Modify: `frontend/src/app/page.tsx`

**Interfaces:**
- Consumes: `getPublicConfig`, `startGoogleLogin`, `validateCallback`, `exchangeCodeForToken`, `getAccessToken`, `clearAuthSession`.
- Produces: static `/login/`, `/auth/callback/` and temporary `/dashboard/` handoff route that Task 5.2 can replace without changing auth storage keys.

- [x] **Step 1: Write failing page behavior tests**
  - Assert login route exposes one button named `Tiếp tục với Google`, no password inputs, and describes the redirect to Cognito/Google.
  - Assert callback renders loading before browser effect, removes OAuth query values with `history.replaceState`, success stores access token then navigates to `/dashboard/`, and errors expose a retry link to `/login/` without leaking query values.
  - Assert dashboard handoff explains that link management is coming from the authenticated session and offers logout, without fake link data.

- [x] **Step 2: Run tests to verify they fail**

Run: `pnpm --dir frontend test`

Expected: FAIL because the routes/components do not exist.

- [x] **Step 3: Implement client boundaries**
  - `GoogleLoginButton` reads config only on click and shows disabled/loading/error state; no automatic redirect during prerender.
  - `AuthCallback` uses `useEffect` and `window.location.search` (tránh phụ thuộc Suspense khi static export), validates state/code/nonce once, xóa query bằng `history.replaceState`, exchange code rồi dùng `window.location.replace("/dashboard/")`.
  - Handle Cognito `error`/`error_description` generically, clear PKCE data after terminal failure, and keep callback query values out of rendered markup.
  - Use semantic headings, `aria-live` for status, and a keyboard reachable retry action.

- [x] **Step 4: Run typecheck and page tests**

Run: `pnpm --dir frontend typecheck && pnpm --dir frontend test`

Expected: PASS; static routes typecheck without Node-only APIs.

### Task 4: Static Build Verification and Project Status

**Files:**
- Modify: `frontend/tests/static-export.test.mjs`
- Modify: `frontend/README.md`
- Modify: `PROJECT_STATUS.md`
- Modify: `task_plan.md`
- Modify: `progress.md`
- Modify: `findings.md`

- [x] **Step 1: Extend static artifact assertions**
  - Check `out/index.html`, `out/login/index.html`, `out/auth/callback/index.html`, `out/dashboard/index.html` and `out/404.html` exist.
  - Check output has product title/copy, references real static assets and contains no starter URLs or password-flow copy.

- [x] **Step 2: Run the frontend verification gate**

Run: `pnpm --dir frontend lint && pnpm --dir frontend typecheck && pnpm --dir frontend test && pnpm --dir frontend build`

Expected: exit 0, all tests pass and `frontend/out/` contains the four routes plus 404/static assets.

- [x] **Step 3: Update status and record evidence**
  - Mark Task 5.1 `complete (implementation/local)` only after the command above passes.
  - Record the exact command/result and known limits: no Terraform apply, CloudFront hosting or Google browser E2E in this task.

- [x] **Step 4: Commit**

```bash
git add docs/superpowers/specs/2026-11-05-phase-5-task-5-1-frontend-design.md docs/superpowers/plans/2026-11-05-phase-5-task-5-1-frontend.md frontend PROJECT_STATUS.md task_plan.md progress.md findings.md
git commit -m "feat: build static Google PKCE frontend pages"
```
