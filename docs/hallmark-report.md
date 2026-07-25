# Hallmark QA Report

- Status: **PASS**
- Web artifact digest (SHA-256): `1baab39305514e8750793462fe26e62141358b1f7c417eeb1785438f9ee0d003`
- Frontend commit: `eb1d7ed425e5a1b1172945afd9af534f49697a1c`
- URL/build: `http://localhost:3000` · Next.js static export served through a local same-origin proxy to the Go API
- Browser/OS/device scale factor: Chrome `150.0.7871.182` headless · Windows 10 · `1`
- Hallmark version: `1.1.0`
- `SKILL.md` SHA-256: `5437b73a1a829210681ca21fa09b7276a5978875f960d3998846e0dc333e9343`
- `slop-test.md` SHA-256: `6eb49fc64ca54929d2c1600e2a6014268f1a7593787c0ed3f6ecb4ac8390ad86`
- Executed at (UTC): `2026-07-25T20:17:28Z`
- Auditor: Codex + Hallmark `audit`

Audit này chạy trên production static export và Go API thật. Ngoài năm viewport
chuẩn, browser đã sweep từng pixel từ `320` đến `1920` CSS px, chạy các luồng
validation, loading, success, conflict `409`, clipboard success/failure, keyboard
focus và reduced motion. Axe kiểm tra cả trạng thái đầu và thành công, không ghi
nhận violation; riêng rule color-contrast có `45` node pass và `0` incomplete.

## Pre-emit critique

| Axis | Score | Evidence | Remaining risk |
|---|---:|---|---|
| Philosophy | 5/5 | Một task surface duy nhất: nhập URL → API thật → copy kết quả. | Không thay thế product usability research. |
| Hierarchy | 5/5 | CTA, form, result và guardrails phân cấp rõ ở mobile lẫn desktop. | Brand hierarchy vẫn phụ thuộc copy hiện tại. |
| Execution | 5/5 | Token, focus, loading delay, state matrix, contrast và width sweep đều có browser evidence. | Chưa kiểm bằng screen reader phần cứng. |
| Specificity | 5/5 | Domain, endpoint, alias/expiration và response metadata đều riêng cho sản phẩm. | Chưa có bộ brand asset chính thức ngoài wordmark. |
| Restraint | 5/5 | Không gradient, fake chrome, feature-card grid hay decoration vô nghĩa. | Không có dark-theme variant để đánh giá. |
| Variety | 4/5 | `Task Surface` với N9 + Ft2 khác generic marketing template. | Chưa có artifact Hallmark trước đó để so knob delta. |

## Viewport evidence

| Viewport | Screenshot | Overflow | Keyboard/state | Result |
|---|---|---:|---|---|
| 320×800 | [`v2-mobile-320.png`](assets/v2-mobile-320.png) | 0 px | CTA không wrap | PASS |
| 375×812 | [`v2-mobile-375.png`](assets/v2-mobile-375.png) | 0 px | Input/label ổn định | PASS |
| 414×896 | [`v2-mobile-414.png`](assets/v2-mobile-414.png) | 0 px | Mobile một cột | PASS |
| 768×1024 | [`v2-tablet-768.png`](assets/v2-tablet-768.png) | 0 px | Field grid đúng breakpoint | PASS |
| 1280×800 | [`v2-desktop-success-1280.png`](assets/v2-desktop-success-1280.png) | 0 px | CTA và result trong first screen | PASS |

Dense responsive sweep: **1,601 widths tested (`320–1920`, bước 1 px), 0
horizontal-overflow violation, 0 clickable-label wrap violation.**

## Interaction and accessibility evidence

| Scenario | Evidence | Result |
|---|---|---|
| Invalid URL | Thông báo cụ thể; focus trở về `#target-url`; không gửi request | PASS |
| Loading chậm | Sau 225 ms: button/input disabled, `aria-busy=true`, spinner hiện; spinner biến mất sau success | PASS |
| Create success | API trả `201`; response envelope được validate; result panel nhận focus | PASS |
| Alias conflict | API trả `409`; copy đúng thông báo domain; result panel nhận focus | PASS |
| Clipboard success | Clipboard chứa đúng `short_url`; status đổi sang “Đã sao chép” | PASS |
| Clipboard denied | Chuyển sang fallback “Sao chép lại” và hướng dẫn copy thủ công | PASS |
| Keyboard | `Ctrl+K` focus URL; tab focus có outline 3 px | PASS |
| Reduced motion | Media query khớp; scroll behavior `auto`; transition rút còn `0.15s` | PASS |
| Axe initial/success | `0` violation ở cả hai state | PASS |
| Color contrast | `45` node pass, `0` incomplete; contrast ví dụ từ `5.31:1` đến `16.87:1` | PASS |
| Console | `0` console error sau khi thêm favicon convention của Next.js | PASS |

## Gate sweep

- Passed: **58/58**
- Failed: **0**
- Unverified: **0**
- Findings: **0 critical · 0 major · 0 minor**

| Gate | Status | Evidence / reason |
|---:|---|---|
| 1 | PASS | Display dùng Space Grotesk, không dùng nhóm font bị cấm. |
| 2 | PASS | Không có gradient hoặc gradient text trong artifact. |
| 3 | PASS | Không có grid ba card bằng nhau. |
| 4 | PASS | Không có card lồng card. |
| 5 | PASS | Không có side-stripe border dày. |
| 6 | PASS | Intro edge-aligned, CTA tách khỏi trục headline. |
| 7 | PASS | Paper/ink là neutral OKLCH có tint, không dùng base `#000/#fff`. |
| 8 | PASS | `Task Surface` form/result không phải Hero → 3 features → CTA. |
| 9 | PASS | Header, workbench, guardrails và footer dùng rule/surface khác nhau. |
| 10 | PASS | Không `transition-all`; chỉ transition thuộc tính có chủ đích. |
| 11 | PASS | Không hover scale. |
| 12 | PASS | Không easing overshoot/bounce. |
| 13 | PASS | Hover control chỉ dùng một tín hiệu transform hoặc underline. |
| 14 | PASS | Motion chỉ dùng `transform`/`opacity`; spinner dùng rotation. |
| 15 | PASS | Keyboard audit xác nhận focus outline xuất hiện tức thì. |
| 16 | PASS | Copy success nằm trong control/result, không toast ăn mừng. |
| 17 | PASS | Không triển khai tooltip. |
| 18 | PASS | Không có nội dung auto-rotate. |
| 19 | PASS | Không có placeholder name/startup cliché. |
| 20 | PASS | Stamp `Task Surface · Cobalt · N9 · Ft2` có ở đầu token artifact. |
| 21 | PASS | Không dùng Specimen. |
| 22 | PASS | Neutral/surface tokens có chroma theo Cobalt. |
| 23 | PASS | Screenshot/computed surface cho thấy accent chỉ ở CTA, link và status có chủ đích. |
| 24 | PASS | Padding/gap/margin lấy từ spacing tokens. |
| 25 | PASS | Prose measure dùng `48ch`, `58ch`, `60ch`. |
| 26 | PASS | Browser đã ép hover/focus/disabled/loading/error/success và copy states. |
| 27 | PASS | Reduced-motion browser run xác nhận fallback hoạt động. |
| 28 | PASS | Artifact không có video cần poster/autoplay treatment. |
| 29 | PASS | Artifact không có abstract background/enrichment giả. |
| 30 | PASS | Không trộn icon library hoặc dùng emoji làm feature icon. |
| 31 | PASS | Không có illustration/Lottie cần accessible treatment. |
| 32 | PASS | Chưa có Hallmark artifact trước đó nên không thể lặp fingerprint cũ; stamp hiện tại là baseline. |
| 33 | PASS | Favicon SVG có vai trò browser icon; page không dùng custom art cần accessible name. |
| 34 | PASS | Sweep từng pixel `320–1920` có 0 overflow. |
| 35 | PASS | Screenshot success xác nhận underline result link không cắt chữ. |
| 36 | PASS | Header/button bars căn giữa, intrinsic items dùng line-height phù hợp. |
| 37 | PASS | Đúng ba family: display, body, mono. |
| 38 | PASS | Mono chỉ ở brand domain và keyboard hint. |
| 38a | PASS | Heading/display đều roman. |
| 39 | PASS | Browser state matrix xác nhận border, helper slot, disabled/loading/error/success. |
| 40 | PASS | Axe color-contrast: 45 node pass, 0 incomplete. |
| 41 | PASS | Axe success-state bao gồm dark result panel và nested foreground swaps. |
| 42 | PASS | N9 edge-aligned header, không phải nav 5 links + CTA. |
| 43 | PASS | Ft2 một dòng, không phải footer bốn cột. |
| 44 | PASS | Screenshot `1280×800` giữ headline, lede, form, CTA và result trong first screen. |
| 45 | PASS | Hero không có decoration vô nghĩa. |
| 46 | PASS | Các con số là giới hạn/HTTP status từ OpenAPI, không phải metric marketing. |
| 47 | PASS | Không vẽ lại browser/terminal/IDE chrome. |
| 48 | PASS | Màu/font ngoài token block tham chiếu semantic token. |
| 49 | PASS | Sweep 1,601 width có 0 clickable-label wrap. |
| 50 | PASS | Không có image-bearing grid. |
| 51 | PASS | Display headings có `min-width: 0` và `overflow-wrap: anywhere`. |
| 52 | PASS | Không có per-theme section-head override. |
| 53 | PASS | Không dùng CSS radio tabs. |
| 54 | PASS | Eyebrow + heading nằm cùng cột dọc. |
| 55 | PASS | Không có uppercase display với line-height dưới `1`. |
| 56 | PASS | Không có sticky element. |
| 57 | PASS | Không có Hallmark `study` diagnosis bị bỏ để quay về catalog theme. |

## Release decision

**PASS — Hallmark visual/interaction release gate đạt 58/58 trên artifact digest
đã ghi ở đầu report.** Đây không phải bằng chứng deploy production, load/SLO,
screen-reader hardware hoặc AWS cutover.
