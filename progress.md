# Progress Log

## Session: 2026-10-26 → 2026-10-27 — Hoàn thiện Phase 0 sau review

- **Status:** complete về đặc tả — Task 0.1 trên nhánh và PR #65 hiện có; PR chờ review/merge. Phase 1 chưa bắt đầu.
- **Phạm vi:** audit đủ 21 bài và source main, sửa đặc tả/kế hoạch, đối chiếu skills, kiểm tra tài liệu và cập nhật trạng thái. Không triển khai Phase 1.
- **Ruling:** Người dùng xác nhận đã xóa AWS cũ; coi đây là baseline do người dùng cung cấp, không lặp lại destroy hoặc khẳng định đã kiểm kê chi phí.
- **Ruling:** Các bản sửa review thuộc Task 0.1 đang mở, tiếp tục nhánh hiện có đã tách từ main; các task mới sau này vẫn tạo branch/PR riêng từ main.
- **Nguồn:** Ba lượt kiểm chứng độc lập đã đọc toàn thân bài 1–7, 8–14 và 15–21 ở hai trang series; source đối chiếu commit `a5ffed9780d0447fc75a42def3693732bdc87907`.
- **Lỗi công cụ đã xử lý:** Lượt review trước xuất nội dung HTML qua Python gặp encoding cp1252; dùng UTF-8 khi đọc nội dung tiếng Việt. Không có thay đổi AWS.
- **Tiếp tục ngày 2026-10-27:** Agent triển khai trước đó bị giới hạn usage và chưa sửa file; đã kiểm tra Git trước khi tiếp tục, giữ nguyên kết quả đọc nguồn. Lỗi mở trang series 2 tạm thời đã được xử lý bằng link phân trang/Python; đủ 21 bài được đọc.
- **Setup đã kiểm:** 314 thư mục skill cấp đầu, 376 `SKILL.md` đệ quy; đủ 13 file đại diện và hai file cấu hình entry hợp lệ.
- **Thay đổi người dùng 2026-10-27:** Chỉ Google login qua Cognito User Pool, Code + PKCE; frontend Hand-Drawn / Doodle & Sketch UI. Đã cập nhật audit, kiến trúc, plan, AGENTS và trạng thái. Google OAuth Web client/secret cần được cung cấp an toàn khi Task 2.2 triển khai; không có secret trong repo.
- **Kiểm tra:** `git diff --check`, Markdown links/anchors, JSON examples và số nguồn 21 bài/21 tracked files; chưa chạy Go/pnpm/Terraform/AWS vì các phần thực thi chưa tồn tại. Reviewer phụ bị giới hạn usage, nên controller tự đối chiếu tài liệu và kiểm tra cấu trúc; không ghi nhận review phụ là hoàn tất.
- **Kiểm tra cụ thể:** Python kiểm 9 Markdown tracked, 20 local links, 1 anchor, 6 JSON blocks, 21 article rows/URLs: PASS. Reference clone ngoài repo đúng SHA và 21 tracked files. `git diff --check`: PASS. Phát hiện `frontend/` untracked scaffold xuất hiện song song; không chỉnh sửa, stage hoặc đưa vào PR Phase 0.
- **PR:** [#65](https://github.com/TrieuNguyenPhu/shorten-link/pull/65) được cập nhật trên cùng branch; chưa merge. Bản sửa hợp đồng/plan ở commit `50044154223a9d587a4559cb2da45a0ebeb6f646`; trạng thái và yêu cầu Google/UI được ghi tiếp trong commit sau.

## Session: 2026-10-26

> Nhật ký lịch sử được giữ lại từ phiên trước. Phiên hoàn thiện không thực hiện lại các thao tác destroy hay xác minh tài khoản. Số file nguồn thực tế ở commit `a5ffed9` là 21, sửa nhận định 19 file bên dưới; mốc đọc toàn thân 21 bài và phạm vi xác minh mới ở mục trên.

### Task 0.1 — Clean Reset, 13 Skill Packs Setup, `AGENTS.md`, Reference Audit & Master Plan
- **Status:** complete
- **Actions taken:**
  1. Kiểm tra và xóa sạch toàn bộ tài nguyên AWS đã triển khai trước đó (`64` resources ở stack `dev` + `5` resources ở stack `bootstrap`). Xác minh bằng AWS CLI (`s3`, `dynamodb`, `lambda`, `apigatewayv2`, `cognito-idp`, `cloudfront`, `logs`) đảm bảo tài khoản sạch 100%.
  2. Xóa toàn bộ file và thư mục cũ tại workspace `d:\source code\shorten-link` (chỉ giữ `.git` để quản lý lịch sử và tạo Pull Request).
  3. Tải và cài đặt toàn bộ 13 bộ skills (`superpowers`, `ponytail`, `planning-with-files`, `caveman`, `trailofbits/skills`, `vercel-labs/agent-skills`, `anthropics/skills`, `gsd-core`, `understand-anything`, `open-code-review`, `ui-ux-pro-max-skill`, `taste-skill`, `claude-code-best-practice`) vào `~/.agents/skills` và cấu hình `.agents/skills.json` + `~/.gemini/config/skills.json`.
  4. Đọc toàn bộ 21 bài viết của series `kkloudtarus.net/blog/series/serverless-on-aws-event-driven` (trang 1 & 2) và toàn bộ 19 file trong repo `nghiadaulau/serverless-url-shortener-aws`.
  5. Thiết lập `AGENTS.md`, `docs/phase-0-reference-audit.md`, `docs/architecture-and-contracts.md`, `docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`, `task_plan.md`, `findings.md`, `progress.md`, `PROJECT_STATUS.md`, và `README.md`.
- **Files created:**
  - `.gitignore`
  - `.agents/skills.json`
  - `AGENTS.md`
  - `docs/phase-0-reference-audit.md`
  - `docs/architecture-and-contracts.md`
  - `docs/superpowers/plans/2026-10-26-serverless-url-shortener-plan.md`
  - `task_plan.md`
  - `findings.md`
  - `progress.md`
  - `PROJECT_STATUS.md`
  - `README.md`
- **Pull Request:** [PR #65](https://github.com/TrieuNguyenPhu/shorten-link/pull/65)

## Session 2026-10-27 — Phase 1
- Task 1.1: in_progress. Đã đọc bốn file trạng thái, plan/spec; không có .codegraph nên dùng tìm kiếm trực tiếp. Branch mới từ main, fast-forward Phase 0; giữ frontend untracked.
- Đã viết test trước implementation: Base62, URL, timestamp/sort, config.
- Task 1.1 complete: RED do GenerateShortCode/NormalizeURL/FormatCreatedAt/Load chưa tồn tại; GREEN toàn bộ unit tests. Windows test/vet PASS; WSL Go1.26.5 + GCC test -race -v và vet PASS. Coverage config 100%, domain 88.2%. Formatting/diff check PASS; review độc lập không có finding.
- Lỗi môi trường: Windows -race thiếu CGO/C compiler; đã cài GCC/libc6-dev trong WSL Ubuntu bằng root và chạy đúng toolchain pin. Không thay cấu hình AWS.
- PR Task 1.1: [#66](https://github.com/TrieuNguyenPhu/shorten-link/pull/66), base Phase 0 (#65); chờ review/merge.
