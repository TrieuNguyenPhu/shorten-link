import Link from "next/link";

import { ShortenerWorkbench } from "@/components/shortener-workbench";

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

        <ShortenerWorkbench />

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
              <dd>URL-only preview, dùng legacy API</dd>
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
