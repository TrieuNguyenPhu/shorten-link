# Quy trình Hallmark QA cho `npt-shortenlink.dev`

> Mục tiêu của tài liệu này là tạo bằng chứng kiểm chứng, không tạo một nhãn “Hallmark pass” mang tính trang trí. **Trạng thái hiện tại của giao diện phải là `NOT RUN` cho đến khi quy trình bên dưới được thực thi trên một commit cụ thể.**

## 1. Nguồn chuẩn và tính tái lập

Quy trình dùng skill cục bộ:

- entrypoint: `.agents/skills/hallmark/SKILL.md`;
- checklist: `.agents/skills/hallmark/references/slop-test.md`;
- audit verb: `.agents/skills/hallmark/references/verbs/audit.md`;
- output contract: `.agents/skills/hallmark/references/contract.md`.

Snapshot khi tài liệu được tạo:

| Thành phần | Giá trị |
|---|---|
| Hallmark version trong front matter | `1.1.0` |
| SHA-256 `SKILL.md` | `5437b73a1a829210681ca21fa09b7276a5978875f960d3998846e0dc333e9343` |
| SHA-256 `slop-test.md` | `6eb49fc64ca54929d2c1600e2a6014268f1a7593787c0ed3f6ecb4ac8390ad86` |

Mỗi báo cáo phải tính lại hash, ghi commit của ứng dụng, browser/version và URL build. Nếu hash khác snapshot thì đọc lại toàn bộ skill/checklist và cập nhật matrix trước khi audit. Không dùng kết quả cũ cho skill mới.

Lưu ý trung thực: README upstream có thể dùng cách đếm khác theo thời điểm. Checklist cục bộ được ghim ở trên có **58 gate thực tế**: số `1–57` cộng gate `38a`. Dự án lấy file cục bộ đã hash làm nguồn chuẩn, không lấy một con số quảng bá trên README làm bằng chứng pass.

Hallmark là skill/verb của agent, **không mặc nhiên là một lệnh npm hay pnpm**. Chỉ ghi `pnpm hallmark:audit` nếu repository thật sự đã định nghĩa script đó; nếu chưa, invocation đúng là yêu cầu agent chạy `hallmark audit <target>` theo skill cục bộ.

## 2. Phạm vi Hallmark

Hallmark kiểm tra visual/interaction layer. Nó không thay thế:

- test business logic hoặc OpenAPI;
- security review và dependency scan;
- performance/load test backend;
- accessibility test đầy đủ với assistive technology;
- quyết định thương hiệu hoặc product copy từ chủ dự án.

Đối với UI này, Hallmark phải giữ chất liệu đã chắt lọc từ bộ ảnh tham khảo ban đầu: editorial kỹ thuật, tương phản xanh đen/sáng, accent xanh lam tiết chế, rule mảnh và thông tin có thứ bậc. Ảnh nguồn đã được xóa khỏi repository; không pixel-clone trang blog, không dựng browser/terminal giả và không invent số liệu, logo hoặc testimonial.

## 3. Cổng pass duy nhất

Một build chỉ được ghi `PASS` khi đồng thời thỏa tất cả:

1. Sáu trục self-critique đều `>= 3/5` sau revision pass.
2. Cả **58/58 gate** có kết quả `PASS`, nghĩa là câu hỏi của gate đều được trả lời “không”.
3. Không gate nào là `N/A` chỉ vì khó kiểm; gate không áp dụng phải có lý do từ chính checklist, còn gate chưa kiểm là `UNVERIFIED` và làm build fail.
4. Hallmark audit cuối có đúng `0 critical · 0 major · 0 minor`.
5. Năm viewport chuẩn và lượt quét liên tục `320–1920` px có bằng chứng.
6. Lint/build/test/a11y liên quan đều pass trên đúng commit được audit.

Không được làm tròn `57/58` thành pass. Không được dùng screenshot desktop để suy ra mobile. Không được tuyên bố “58/58” trước khi build và gate sweep hoàn tất.

## 4. Sáu trục pre-emit self-critique

Chấm trước gate sweep. Bất kỳ trục nào dưới `3` buộc sửa và chấm lại.

| Mã | Trục | Câu hỏi kiểm chứng | Ngưỡng |
|---|---|---|---:|
| P | Philosophy | Giao diện có lập trường phục vụ tác vụ rút gọn link, hay chỉ là một layout đẹp? | `>= 3` |
| H | Hierarchy | Trong 2 giây có phân biệt được hành động chính, thông tin phụ và tertiary content? | `>= 3` |
| E | Execution | Token, wrap, focus, contrast, alignment và responsive có hoàn thiện ở chi tiết? | `>= 3` |
| S | Specificity | Đây có rõ là `npt-shortenlink.dev`, hay có thể thay logo thành bất kỳ SaaS nào? | `>= 3` |
| R | Restraint | Mọi decoration/motion/card có thực sự kiếm được vị trí của nó? | `>= 3` |
| V | Variety | Cấu trúc có khác fingerprint Hallmark gần nhất, không chỉ đổi màu? | `>= 3` |

Stamp bắt buộc ở đầu artifact/style theo skill:

```css
/* Hallmark · pre-emit critique: P4 H5 E4 S4 R5 V4 */
```

Điểm là đánh giá có lý do, không phải mục tiêu để chấm toàn `5`. Báo cáo phải ghi một câu bằng chứng và một rủi ro còn lại cho từng trục.

## 5. Bản đồ 58 gate

| Nhóm | Gate | Số gate | Bằng chứng tối thiểu |
|---|---:|---:|---|
| Visual | `1–7` | 7 | computed styles, screenshot, font/token inspection |
| Structural | `8–9` | 2 | DOM outline, screenshot toàn trang, `.hallmark/log.json`/stamp nếu có |
| Microinteractions | `10–19` | 10 | CSS search, hover/focus/keyboard, reduced-motion test |
| Variety | `20–21` | 2 | CSS stamp và lịch sử Hallmark |
| Implementation | `22–27` | 6 | token/spacing audit, state matrix, motion fallback |
| Hero enrichment | `28–31` | 4 | asset/DOM inspection; ghi rõ nếu trang thực sự không có enrichment |
| Diversification/a11y art | `32–33` | 2 | knob delta/stamp, SVG/CSS-art accessible name |
| Layout safety | `34–36` | 3 | width sweep, decorative position, flex alignment |
| Typography discipline | `37`, `38`, `38a` | 3 | font-family inventory và heading computed style |
| Input state | `39` | 1 | đo border/outline/height/helper slot/disabled channels |
| Contrast/readability | `40–41` | 2 | WCAG/APCA report theo computed foreground/background |
| Nav/footer/hero structure | `42–45` | 4 | screenshot, DOM structure, hero fold `1280×800` |
| Honest copy | `46` | 1 | provenance cho mọi metric/claim hoặc xác nhận không có metric |
| Re-drawn chrome | `47` | 1 | DOM/asset inspection |
| Token discipline | `48` | 1 | tìm raw color/font ngoài token block |
| Clickable responsiveness | `49` | 1 | mọi CTA/nav/footer/tab/breadcrumb ở width sweep |
| Mobile non-negotiables | `50–57` | 8 | grid/wrap/theme head/tab/sticky/DNA checks ở mobile |
| **Tổng** |  | **58** | tất cả có evidence và owner |

Checklist chi tiết không được chép lại bằng trí nhớ; auditor phải mở file `slop-test.md` đã hash và đi lần lượt từng gate. Bảng trên chỉ là index để quản lý bằng chứng.

## 6. Viewport matrix bắt buộc

Hallmark bắt buộc các width `320`, `375`, `414`, `768` px và desktop `1280×800`. Dự án chọn height chụp chuẩn cho bốn width đầu để artifact tái lập; height không thay thế lượt quét liên tục.

| ID | Viewport chụp | Mục tiêu chính |
|---|---:|---|
| `mobile-320` | `320×800` | trường hợp hẹp nhất, CTA không wrap, không overflow |
| `mobile-375` | `375×812` | mobile phổ biến, input/button và helper slot ổn định |
| `mobile-414` | `414×896` | mobile rộng, không vô tình giữ desktop columns |
| `tablet-768` | `768×1024` | breakpoint tablet, section head và grid collapse đúng |
| `laptop-fold` | `1280×800` | toàn bộ hero thiết yếu + primary CTA thấy được không cuộn |

Ngoài năm ảnh, kéo/rescale liên tục từ `320` tới `1920` CSS px để bắt overflow ở width trung gian. Bắt buộc:

- `overflow-x: clip` trên cả `html` và `body`, không dùng `hidden` để che lỗi;
- không có clickable label hai dòng;
- display heading có `overflow-wrap: anywhere` và `min-width: 0`;
- grid chứa ảnh dùng `minmax(0, 1fr)`;
- section head có eyebrow + heading phải xếp dọc;
- sticky con nằm dưới sticky nav, không cùng `top: 0`;
- tại `1280×800`, headline, lede, form/CTA và focal point chính nằm trong first screen.

## 7. Ma trận trạng thái tương tác

Mọi control tương tác phải được kiểm bằng hành vi thật hoặc preview cưỡng bức state, không chỉ đọc CSS:

| State | Input URL | Alias/expiration | Submit | Copy result |
|---|---|---|---|---|
| default | hiển thị label/value rõ | layout ổn định | enabled khi hợp lệ | có accessible name |
| hover | một tín hiệu thị giác | một tín hiệu | không ghép nhiều effect | một tín hiệu |
| focus-visible | ring tức thì, đủ contrast | ring tức thì | ring tức thì | ring tức thì |
| active | không đổi geometry | không đổi geometry | feedback ngắn | feedback ngắn |
| disabled | opacity + cursor + native/ARIA | nếu có | không submit | không copy |
| loading | giữ chiều cao/helper slot | không nhảy layout | label/progress rõ | không áp dụng nếu chưa có result |
| error | border-width vẫn 1 px, helper reserved | field error đúng | không toast ăn mừng | copy failure có message |
| success | không xóa dữ liệu ngoài ý muốn | giữ context | result hiển thị | xác nhận im lặng trong ngữ cảnh |

`focus-visible` không animate vào. Spatial motion chỉ animate `transform`/`opacity`, có `prefers-reduced-motion`; fallback tối đa là crossfade ngắn theo skill.

## 8. Quy trình thực thi

### Bước 0 — Ghim subject

Ghi vào report:

- Git commit SHA/digest của web artifact;
- URL local/staging chính xác;
- Hallmark version và SHA-256 hai file nguồn;
- browser engine/version, OS, device scale factor;
- mode/theme nếu có và đường dẫn CSS stamp.

Nếu working tree thay đổi trong khi audit, dừng hoặc tạo report mới; không trộn bằng chứng của hai revision.

### Bước 1 — Hallmark pre-flight

Đọc code trước khi đánh giá: `design.md` nếu có, package/framework, font, palette/token, motion dependency, spacing và global stylesheet. Ghi rõ thứ gì được giữ, thứ gì được Hallmark giới thiệu. Không ghi “không có signal” nếu chưa scan.

Kiểm tra design intent từ ảnh tham khảo:

- genre dự kiến: modern-minimal/editorial kỹ thuật;
- primary action: tạo short link;
- tone: technical, austere, tin cậy;
- accent phải tiết chế, không gradient text;
- không invent metric, testimonial, customer logo hoặc uptime claim.

Nếu implementer chọn genre/theme khác, phải ghi lý do và audit theo genre đã stamp; không đổi rule giữa chừng để hợp thức hóa lỗi.

### Bước 2 — Kiểm tra tĩnh trước khi render

1. Chạy lint/typecheck/build/test bằng script thật trong repository.
2. Tìm `transition-all`, raw color/font ngoài token, hover-scale lặp, italic heading, fake chrome và emoji-as-feature-icon.
3. Inventory font family (`<= 3`) và outlier slots (`<= 2`).
4. Kiểm tra state styles, reduced motion, semantic label/ARIA và error live region.
5. Kiểm tra mọi product claim có nguồn; claim chưa xác nhận phải bỏ hoặc đánh dấu placeholder rõ ràng.

Kết quả search có phát hiện phải dẫn `file:line`; không kết luận pass từ việc một regex không match nếu gate cần render.

### Bước 3 — Render và automation

1. Build/start đúng artifact sẽ release.
2. Dùng browser automation chụp full-page và first-screen ở năm viewport.
3. Chạy keyboard flow: nhập URL → submit → đọc result → copy; chạy cả validation/API error.
4. Chạy accessibility scan tự động và contrast calculation trên computed color/background.
5. Test reduced motion, zoom/reflow và request failure/slow response.
6. Lưu screenshot, trace/video khi fail và output console/network có liên quan.

Automation hỗ trợ bằng chứng nhưng không tự quyết các gate về specificity, restraint, structural fingerprint hoặc invented copy; các mục đó cần review có lý do.

### Bước 4 — Self-critique và revision

Chấm sáu trục. Nếu một trục `< 3`, sửa trước gate sweep rồi chụp lại evidence bị ảnh hưởng. Không nâng điểm mà không thay đổi artifact hoặc bổ sung bằng chứng.

### Bước 5 — Sweep đủ 58 gate

Mở checklist cục bộ và ghi từng gate:

```text
Gate 34 · PASS · screenshots/mobile-320.png + width-sweep.webm
Gate 40 · PASS · reports/contrast.json
Gate 44 · PASS · screenshots/1280x800-first-screen.png
Gate 46 · PASS · no quantitative product claims; copy reviewed at <commit>
```

Trạng thái hợp lệ: `PASS`, `FAIL`, `UNVERIFIED`. Chỉ dùng `NOT_APPLICABLE` khi wording/genre note trong gate thực sự cho phép, kèm lý do; `NOT_APPLICABLE` không được dùng để bỏ qua một yêu cầu universal.

Nếu bất kỳ gate nào fail, sửa, chạy lại gate liên quan và regression sweep các viewport; không chỉ sửa file report.

### Bước 6 — Chạy `hallmark audit`

Invocation:

```text
hallmark audit apps/web/src/app
```

Audit là read-only: trả từng finding gồm **Tell**, **Where**, **Severity**, **Fix**, nhóm theo `critical/major/minor`, và kết thúc đúng format `N critical · M major · K minor`. Audit phải kiểm cả structural fingerprint và stamp-vs-page; stamp nói một macrostructure nhưng DOM/render không khớp là `critical: stamp lies`.

Sau khi sửa finding, chạy audit lại trên commit mới. Không sửa lén trong chính audit pass và không giữ count cũ.

### Bước 7 — Ký report

Report cuối ghi decision `PASS` hoặc `FAIL`, người/agent chạy, thời điểm UTC, commit và link evidence. Một report thiếu evidence hoặc có `UNVERIFIED` luôn là `FAIL`, kể cả summary ghi `58/58`.

## 9. Template báo cáo

```markdown
# Hallmark QA Report

- Status: PASS | FAIL | NOT RUN
- App commit/artifact digest:
- URL/build command:
- Browser/OS/device scale factor:
- Hallmark version:
- SKILL.md SHA-256:
- slop-test.md SHA-256:
- Executed at (UTC):
- Auditor:

## Pre-emit critique
| Axis | Score | Evidence | Remaining risk |
|---|---:|---|---|
| Philosophy | /5 | | |
| Hierarchy | /5 | | |
| Execution | /5 | | |
| Specificity | /5 | | |
| Restraint | /5 | | |
| Variety | /5 | | |

## Viewports
| Viewport | Screenshot | Overflow sweep | Keyboard | Result |
|---|---|---|---|---|
| 320×800 | | | | |
| 375×812 | | | | |
| 414×896 | | | | |
| 768×1024 | | | | |
| 1280×800 | | | | |

## Gate sweep
- Passed: /58
- Failed:
- Unverified:
- Not applicable with rule-based reason:

| Gate | Status | Evidence | Finding/fix |
|---:|---|---|---|
| 1 | | | |
| ... | | | |
| 38a | | | |
| ... | | | |
| 57 | | | |

## Audit findings
| Tell | Where | Severity | Fix | Resolution commit |
|---|---|---|---|---|

0 critical · 0 major · 0 minor

## Release decision
PASS chỉ khi axes >=3, 58/58 có evidence, không UNVERIFIED,
audit 0/0/0 và mọi viewport/check liên quan pass.
```

## 10. Điều kiện chặn release

Release bị chặn ngay khi có một trong các mục:

- bất kỳ self-critique axis `< 3`;
- gate `FAIL` hoặc `UNVERIFIED`;
- audit khác `0 critical · 0 major · 0 minor`;
- horizontal scroll ở bất kỳ width nào trong `320–1920`;
- CTA/nav/footer link wrap hai dòng;
- hero thiết yếu không nằm trong `1280×800` first screen;
- contrast/focus/input state không đạt;
- product metric/claim không có nguồn;
- screenshot/report thuộc commit khác artifact release;
- report chỉ nêu kết quả mà không có evidence có thể truy ngược.

Không có ngoại lệ bằng lời nói “trông ổn trên máy tôi”. Nếu cần ship khẩn cấp, decision vẫn là `FAIL` và exception phải nằm trong release record với owner/risk; không đổi report thành `PASS`.

## 11. Tài liệu tham khảo

- [`Nutlope/hallmark`](https://github.com/Nutlope/hallmark): upstream của design skill. Checklist cục bộ đã ghim/hash mới là nguồn trực tiếp cho audit của repository này.
- [`nghiadaulau/serverless-url-shortener-aws`](https://github.com/nghiadaulau/serverless-url-shortener-aws): nguồn tham khảo cho bối cảnh sản phẩm/AWS; không phải bằng chứng giao diện đã vượt Hallmark.
