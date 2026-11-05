# Shrnk Frontend Updates Design

## Intent
Đổi nhận diện website thành Shrnk và làm rõ trải nghiệm signed-in: logo dùng icon link, CTA gọn, dashboard là nơi xem dữ liệu, còn trang chủ hiển thị form tạo link khi có phiên đăng nhập.

## Goals
- Logo/banner dùng cùng icon link; tên sản phẩm là Shrnk.
- Xóa mũi tên khỏi nút và các dòng chú thích nhỏ dưới đối tượng thông tin.
- Navbar nhận biết phiên đăng nhập phía client; người đã đăng nhập thấy Dashboard/Đăng xuất và nhấn logo vẫn về trang chủ với trạng thái đúng.
- Đổi CTA login thành Source Code, trỏ tới repository GitHub.
- Xóa route My Links khỏi điều hướng; dashboard không còn form tạo link.
- Đổi CTA thành Kiến trúc dự án và đưa tới route `/architecture/`.
- Trang chủ hiện form tạo link dưới phần giới thiệu khi có access token hợp lệ.

## Approach
Giữ static export và API hiện tại, dùng component client nhỏ cho Navbar và form tạo link trên landing. Dashboard tiếp tục dùng LinksWorkspace nhưng ẩn CreateLinkForm. Thêm route kiến trúc tĩnh và test source/static để khóa copy, route và trạng thái auth.

## Constraints
Không thay đổi backend/API, không thêm dependency, không dùng SSR/server actions, giữ Doodle UI và Google PKCE.
