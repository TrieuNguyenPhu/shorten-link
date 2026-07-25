import Link from "next/link";

export default function Home() {
  return (
    <div className="site-shell">
      <header className="site-header">
        <Link className="brand" href="/" aria-label="Trang chủ NPT ShortenLink">
          <span className="brand-mark" aria-hidden="true">
            npt/
          </span>
          <span className="brand-domain">shortenlink</span>
        </Link>
        <span className="preview-status">Frontend preview</span>
      </header>

      <main className="site-main">
        <section className="intro" aria-labelledby="page-title">
          <div className="intro__title">
            <p className="intro__context">Bản xem trước · chưa production</p>
            <h1 id="page-title">
              Rút gọn URL. Giữ nguyên nền tảng hiện tại.
            </h1>
          </div>
          <p className="intro__lede">
            Giao diện này là hướng thử nghiệm cho NPT ShortenLink và sẽ dùng
            AWS Lambda viết bằng Python cùng API hiện có. Ứng dụng React/Vite
            vẫn là frontend production.
          </p>
        </section>

        <section className="task-surface" aria-labelledby="surface-title">
          <div className="task-surface__copy">
            <h2 id="surface-title">Bề mặt mới. Hợp đồng cũ còn nguyên.</h2>
            <p>
              Bản xem trước này chỉ thiết lập visual shell. Form và kết nối API
              sẽ được bổ sung trong các thay đổi riêng sau khi qua compatibility
              gate.
            </p>
          </div>

          <ol className="contract-flow" aria-label="Luồng rút gọn URL hiện tại">
            <li>
              <span className="contract-flow__index">01</span>
              <div>
                <h3>URL đầu vào</h3>
                <p>Request chỉ mang URL cần rút gọn.</p>
              </div>
            </li>
            <li>
              <span className="contract-flow__index">02</span>
              <div>
                <h3>Python Lambda</h3>
                <p>API hiện tại tiếp tục chạy trên AWS SAM.</p>
              </div>
            </li>
            <li>
              <span className="contract-flow__index">03</span>
              <div>
                <h3>Mã rút gọn</h3>
                <p>Response hiện tại trả về short_url_code.</p>
              </div>
            </li>
          </ol>
        </section>

        <section className="compatibility" aria-labelledby="compatibility-title">
          <div className="compatibility__heading">
            <h2 id="compatibility-title">Giới hạn của preview này.</h2>
            <p>Không thay thế, không deploy, không đổi production route.</p>
          </div>
          <dl className="compatibility-list">
            <div>
              <dt>Production frontend</dt>
              <dd>React/Vite hiện tại</dd>
            </div>
            <div>
              <dt>Backend</dt>
              <dd>Python Lambda qua AWS SAM</dd>
            </div>
            <div>
              <dt>Trạng thái</dt>
              <dd>Visual preview, chưa gọi API</dd>
            </div>
          </dl>
        </section>
      </main>

      <footer className="site-footer">
        <p>NPT ShortenLink · Frontend preview · Python + AWS SAM</p>
      </footer>
    </div>
  );
}
