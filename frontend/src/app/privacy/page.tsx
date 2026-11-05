import type { Metadata } from "next";
import Link from "next/link";
import { LinkIcon } from "@/components/LinkIcon";
import { PRIVACY_POLICY_VERSION } from "@/lib/auth";

export const metadata: Metadata = { title: "Chính sách bảo vệ dữ liệu cá nhân" };

export default function PrivacyPolicyPage() {
  return (
    <main className="mx-auto w-full max-w-4xl flex-1 px-5 pb-16 pt-10 sm:px-8 sm:pt-16" id="main-content">
      <section className="sketch-card space-y-6 leading-relaxed">
        <div className="flex items-center gap-4">
          <LinkIcon className="h-14 w-14" />
          <div>
            <h1 className="font-display text-3xl font-bold sm:text-4xl">Chính sách bảo vệ dữ liệu cá nhân</h1>
            <p className="mt-1 text-sm text-ink-muted">
              Cập nhật lần cuối: 05/11/2026 · Phiên bản: {PRIVACY_POLICY_VERSION}
            </p>
          </div>
        </div>

        <p className="rounded-xl border-2 border-dashed border-coral/60 bg-sun/30 p-4 text-sm">
          <strong>BẢN NHÁP</strong> do công cụ tạo từ mã nguồn website ngày 05/11/2026 theo khung Luật Bảo vệ dữ liệu cá nhân số 91/2025/QH15 và Nghị định 356/2025/NĐ-CP. Cần người có chuyên môn pháp lý xem lại và điền đầy đủ các mục <code>[ĐIỀN: ...]</code> trước khi vận hành chính thức.
        </p>

        {/* Căn cứ: điểm b khoản 2 Điều 9 Luật 91/2025/QH15: người dùng phải biết rõ bên kiểm soát dữ liệu khi đồng ý. */}
        <section className="space-y-2">
          <h2 className="font-display text-2xl font-bold">1. Chúng tôi là ai</h2>
          <p>
            <strong>[ĐIỀN: Tên đơn vị / cá nhân vận hành Shrnk]</strong> (sau đây gọi là &ldquo;chúng tôi&rdquo;) là bên kiểm soát và xử lý dữ liệu cá nhân được thu thập qua ứng dụng rút gọn liên kết <strong>Shrnk</strong>.
          </p>
          <ul className="list-disc space-y-1 pl-6 text-ink-muted">
            <li>Mã số doanh nghiệp / hộ kinh doanh (nếu có): <code>[ĐIỀN]</code></li>
            <li>Địa chỉ liên hệ: <code>[ĐIỀN]</code></li>
            <li>Email tiếp nhận yêu cầu về dữ liệu cá nhân: <code>[ĐIỀN: email liên hệ]</code></li>
          </ul>
        </section>

        {/* Căn cứ: Điều 9 khoản 2 điểm a và khoản 4 điểm a Luật 91/2025/QH15. */}
        <section className="space-y-3">
          <h2 className="font-display text-2xl font-bold">2. Dữ liệu chúng tôi thu thập và mục đích</h2>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[540px] border-collapse text-left text-sm">
              <thead>
                <tr className="border-b-2 border-ink bg-paper-deep/50">
                  <th className="p-3 font-bold">Loại dữ liệu (cơ bản)</th>
                  <th className="p-3 font-bold">Khi nào thu thập</th>
                  <th className="p-3 font-bold">Mục đích xử lý</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ink/15">
                <tr>
                  <td className="p-3">Địa chỉ email, mã định danh tài khoản (<code>sub</code>)</td>
                  <td className="p-3">Khi bạn đồng ý và đăng nhập bằng Google OAuth tại trang <code>/login/</code></td>
                  <td className="p-3">Xác thực người dùng và phân quyền quản lý các liên kết rút gọn do bạn tạo</td>
                </tr>
                <tr>
                  <td className="p-3">URL đích, mã rút gọn (<code>code</code>), thời điểm tạo, bộ đếm lượt truy cập (<code>clicks</code>)</td>
                  <td className="p-3">Khi bạn tạo liên kết mới hoặc khi có người mở liên kết rút gọn</td>
                  <td className="p-3">Chuyển hướng liên kết rút gọn và hiển thị số lượt click theo thời gian thực trên Dashboard</td>
                </tr>
                <tr>
                  <td className="p-3">Nhật ký kỹ thuật (mã yêu cầu, đường dẫn API, mã trạng thái HTTP)</td>
                  <td className="p-3">Khi trình duyệt gọi API hoặc kết nối WebSocket</td>
                  <td className="p-3">Vận hành kỹ thuật, xử lý sự cố và bảo đảm an ninh mạng</td>
                </tr>
              </tbody>
            </table>
          </div>
          <p className="text-sm text-ink-muted">
            Chúng tôi chỉ thu thập dữ liệu cá nhân cơ bản nêu trên, không thu thập dữ liệu cá nhân nhạy cảm và không dùng dữ liệu cho mục đích khác nếu chưa xin ý kiến đồng ý mới của bạn.
          </p>
        </section>

        {/* Căn cứ: Điều 7 khoản 6 Luật 91/2025/QH15 cấm mua bán dữ liệu cá nhân. */}
        <section className="space-y-3">
          <h2 className="font-display text-2xl font-bold">3. Bên thứ ba nhận dữ liệu</h2>
          <ul className="list-disc space-y-1 pl-6">
            <li>
              <strong>Google LLC (Google OAuth 2.0):</strong> Xác thực tài khoản Google khi bạn chọn đăng nhập và cung cấp địa chỉ email cho hệ thống xác thực.
            </li>
            <li>
              <strong>Amazon Web Services (AWS):</strong> Cung cấp hạ tầng điện toán đám mây lưu trữ và xử lý dữ liệu (Amazon Cognito, Amazon DynamoDB, AWS Lambda, Amazon API Gateway, Amazon EventBridge, Amazon SQS, Amazon S3 và Amazon CloudFront).
            </li>
          </ul>
          <p>Chúng tôi không bán dữ liệu cá nhân của bạn cho bất kỳ bên thứ ba nào.</p>
        </section>

        {/* Căn cứ: Điều 20 khoản 1 và Điều 14 khoản 1 điểm b Luật 91/2025/QH15. */}
        <section className="space-y-2">
          <h2 className="font-display text-2xl font-bold">4. Lưu trữ và chuyển dữ liệu ra nước ngoài</h2>
          <p>
            Toàn bộ dữ liệu tài khoản và liên kết rút gọn được lưu trữ trên hạ tầng <strong>Amazon Web Services (AWS) đặt tại Singapore (region <code>ap-southeast-1</code>)</strong>. Việc lưu trữ này thuộc trường hợp chuyển dữ liệu cá nhân xuyên biên giới theo khoản 1 Điều 20 Luật Bảo vệ dữ liệu cá nhân số 91/2025/QH15.
          </p>
          <p>
            <strong>Thời hạn lưu trữ:</strong> Liên kết rút gọn được lưu cho đến khi bạn chủ động xóa trên Dashboard hoặc yêu cầu xóa tài khoản; bản ghi chống trùng sự kiện click tự động xóa sau 24 giờ; trạng thái kết nối WebSocket tự động xóa sau 2 giờ; nhật ký kỹ thuật trên AWS CloudWatch được lưu tối đa theo cấu hình vận hành (<code>[KIỂM TRA: 14 ngày trên môi trường dev / tối thiểu 12 tháng khi vận hành chính thức theo NĐ 333/2026/NĐ-CP]</code>).
          </p>
        </section>

        {/* Căn cứ: Điều 9 khoản 4 điểm d, khoản 8 Điều 28 Luật 91/2025/QH15. */}
        <section className="space-y-2">
          <h2 className="font-display text-2xl font-bold">5. Bộ nhớ trình duyệt và công cụ theo dõi</h2>
          <p>
            Shrnk <strong>không</strong> sử dụng Google Analytics, Meta Pixel, TikTok Pixel hay bất kỳ cookie quảng cáo, theo dõi hành vi nào.
          </p>
          <p>
            Trình duyệt của bạn chỉ lưu tạm thời trong <code>sessionStorage</code> các khóa cần thiết cho phiên đăng nhập hiện tại (mã truy cập <code>access_token</code>, thời hạn hết hạn, mã kiểm chứng PKCE và thời điểm đồng ý chính sách). Các dữ liệu này tự động bị xóa ngay khi bạn bấm &ldquo;Đăng xuất&rdquo; hoặc đóng thẻ trình duyệt.
          </p>
        </section>

        {/* Căn cứ: Điều 4 khoản 1, Điều 10 khoản 2 Luật 91/2025/QH15; Điều 5 khoản 2, 3, 4 Nghị định 356/2025/NĐ-CP. */}
        <section className="space-y-2">
          <h2 className="font-display text-2xl font-bold">6. Quyền của bạn và thời hạn xử lý yêu cầu</h2>
          <p>
            Theo khoản 1 Điều 4 Luật Bảo vệ dữ liệu cá nhân số 91/2025/QH15, bạn có quyền được biết, đồng ý hoặc rút lại sự đồng ý, xem, chỉnh sửa hoặc yêu cầu chỉnh sửa, yêu cầu cung cấp, xóa, hạn chế xử lý, phản đối xử lý dữ liệu cá nhân, cũng như khiếu nại, tố cáo, khởi kiện và yêu cầu bồi thường theo quy định pháp luật.
          </p>
          <p>
            Bạn có thể xóa trực tiếp từng liên kết rút gọn tại trang Dashboard hoặc gửi yêu cầu về tài khoản/dữ liệu cá nhân tới email <code>[ĐIỀN: email liên hệ]</code>. Theo Điều 5 Nghị định 356/2025/NĐ-CP, chúng tôi thực hiện trong các thời hạn sau:
          </p>
          <ul className="list-disc space-y-1 pl-6">
            <li>Xác nhận đã nhận yêu cầu và hướng dẫn thủ tục: trong <strong>2 ngày làm việc</strong>.</li>
            <li>Cho xem, chỉnh sửa hoặc cung cấp dữ liệu cá nhân: trong <strong>10 ngày</strong> (<strong>15 ngày</strong> nếu cần bên xử lý dữ liệu hoặc bên thứ ba cùng thực hiện).</li>
            <li>Xử lý yêu cầu rút lại sự đồng ý, hạn chế hoặc phản đối xử lý dữ liệu: trong <strong>15 ngày</strong> (<strong>20 ngày</strong> nếu cần bên xử lý dữ liệu hoặc bên thứ ba ngừng xử lý).</li>
            <li>Xóa dữ liệu cá nhân: trong <strong>20 ngày</strong> (<strong>30 ngày</strong> nếu cần bên xử lý dữ liệu hoặc bên thứ ba cùng xóa).</li>
          </ul>
          <p className="text-sm text-ink-muted">
            Trường hợp yêu cầu phức tạp cần gia hạn (tối đa 1 lần: 10 ngày đối với xem/sửa/cung cấp, 15 ngày đối với rút đồng ý/hạn chế/phản đối, 20 ngày đối với xóa dữ liệu), chúng tôi sẽ thông báo rõ lý do cho bạn trong thời hạn ban đầu.
          </p>
        </section>

        {/* Căn cứ: Điều 23 khoản 1 Luật 91/2025/QH15. */}
        <section className="space-y-2">
          <h2 className="font-display text-2xl font-bold">7. Khi có sự cố và thay đổi chính sách</h2>
          <p>
            Nếu phát hiện hành vi vi phạm hoặc sự cố rò rỉ, mất dữ liệu cá nhân, chúng tôi sẽ thông báo cho cơ quan chuyên trách bảo vệ dữ liệu cá nhân chậm nhất <strong>72 giờ</strong> theo khoản 1 Điều 23 Luật 91/2025/QH15. Khi thay đổi mục đích hoặc phạm vi xử lý dữ liệu cá nhân, chúng tôi sẽ cập nhật phiên bản chính sách trên trang này và xin lại sự đồng ý của bạn trước khi áp dụng.
          </p>
        </section>

        <div className="pt-4">
          <Link className="sketch-button sketch-button-secondary inline-flex" href="/login/">
            Quay lại trang đăng nhập
          </Link>
        </div>
      </section>
    </main>
  );
}
