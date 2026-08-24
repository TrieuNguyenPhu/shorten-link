import Link from "next/link";

import { ShortenerWorkbench } from "@/components/shortener-workbench";

export default function Home() {
  return (
    <div className="site-shell">
      <a className="skip-link" href="#main-content">
        Bỏ qua đến nội dung chính
      </a>
      <header className="site-header">
        <Link className="brand" href="/" aria-label="Trang chủ NPT ShortenLink">
          <span className="brand-mark" aria-hidden="true">
            <svg viewBox="0 0 24 24" role="presentation">
              <path d="M9.5 14.5 14.5 9M7.2 17.8l-1 .9a3.5 3.5 0 0 1-5-5l3.3-3.2a3.5 3.5 0 0 1 5 0M16.8 6.2l1-.9a3.5 3.5 0 0 1 5 5l-3.3 3.2a3.5 3.5 0 0 1-5 0" />
            </svg>
          </span>
          <span className="brand-domain">
            <strong>NPT</strong>
            <span>ShortenLink</span>
          </span>
        </Link>
        <div className="header-meta">
          <span className="header-note">OpenAPI-first</span>
          <a className="header-action" href="#shorten-form">
            Tạo link
            <svg viewBox="0 0 20 20" aria-hidden="true">
              <path d="m7 5 5 5-5 5" />
            </svg>
          </a>
        </div>
      </header>

      <main id="main-content" className="site-main">
        <section className="intro" aria-labelledby="page-title">
          <div className="intro__copy">
            <p className="intro__eyebrow">
              <span aria-hidden="true" />
              Trình rút gọn URL tối giản
            </p>
            <h1 id="page-title">
              Link dài đi vào.
              <span> Link gọn đi ra.</span>
            </h1>
            <p className="intro__lede">
              Tạo đường dẫn dễ chia sẻ với alias và thời hạn theo ý bạn. Không
              tài khoản, không bước thừa.
            </p>
            <ul className="intro__benefits" aria-label="Tính năng chính">
              <li>Alias tùy chọn</li>
              <li>Hết hạn linh hoạt</li>
              <li>Sao chép tức thì</li>
            </ul>
          </div>

          <div
            className="intro__preview"
            role="img"
            aria-label="Ví dụ một URL dài được rút gọn thành npt-shortenlink.dev/link/mua-he"
          >
            <div className="preview-line">
              <span>URL gốc</span>
              <p>example.com/tai-lieu/chien-dich-mua-he-2026</p>
            </div>
            <div className="preview-flow" aria-hidden="true">
              <span />
              <svg viewBox="0 0 20 20">
                <path d="m7 5 5 5-5 5" />
              </svg>
            </div>
            <div className="preview-line preview-line--result">
              <span>Short link</span>
              <p>npt-shortenlink.dev/link/mua-he</p>
            </div>
          </div>
        </section>

        <ShortenerWorkbench />

        <section className="guardrails" aria-labelledby="guardrails-title">
          <div className="guardrails__heading">
            <p className="section-kicker">Thông số rõ ràng</p>
            <h2 id="guardrails-title">Mọi thứ bạn cần. Không hơn.</h2>
            <p>
              Frontend và API cùng tuân theo một contract OpenAPI, để link tạo
              ra luôn đúng như bạn mong đợi.
            </p>
          </div>
          <dl className="spec-list">
            <div>
              <dt>URL hợp lệ</dt>
              <dd>HTTP hoặc HTTPS, tối đa 2.048 ký tự.</dd>
            </div>
            <div>
              <dt>Alias của riêng bạn</dt>
              <dd>4–32 ký tự thường, chữ số hoặc gạch ngang.</dd>
            </div>
            <div>
              <dt>Thời hạn linh hoạt</dt>
              <dd>Từ 1–365 ngày, hoặc để trống nếu không cần.</dd>
            </div>
            <div>
              <dt>Chuyển hướng an toàn</dt>
              <dd>HTTP 302, không cache đích đến vĩnh viễn.</dd>
            </div>
          </dl>
        </section>
      </main>

      <footer className="site-footer">
        <Link className="footer-brand" href="/">
          NPT ShortenLink
        </Link>
        <p>Go + Next.js · AWS SAM</p>
        <a href="#shorten-form">Trở lại tạo link</a>
      </footer>
    </div>
  );
}
