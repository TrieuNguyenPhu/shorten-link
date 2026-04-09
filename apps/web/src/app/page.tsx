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
          <span className="brand-domain">shortenlink.dev</span>
        </Link>
        <a className="header-action" href="#shorten-form">
          Tạo link ↓
        </a>
      </header>

      <main className="site-main">
        <section className="intro" aria-labelledby="page-title">
          <div>
            <p className="intro__domain">npt-shortenlink.dev</p>
            <h1 id="page-title">Link ngắn. Quyền kiểm soát nguyên vẹn.</h1>
          </div>
          <p className="intro__lede">
            Dán URL HTTP hoặc HTTPS, chọn alias và thời hạn nếu cần. Hệ thống
            trả về một đường dẫn sẵn để sao chép.
          </p>
        </section>

        <ShortenerWorkbench />

        <section className="guardrails" aria-labelledby="guardrails-title">
          <div className="guardrails__heading">
            <h2 id="guardrails-title">Quy tắc rõ trước khi tạo.</h2>
            <p>Frontend và API cùng tuân theo một contract OpenAPI.</p>
          </div>
          <dl className="spec-list">
            <div>
              <dt>URL</dt>
              <dd>HTTP hoặc HTTPS</dd>
              <dd>Tối đa 2.048 ký tự</dd>
            </div>
            <div>
              <dt>Alias</dt>
              <dd>4–32 ký tự</dd>
              <dd>Chữ thường, số, gạch ngang</dd>
            </div>
            <div>
              <dt>Thời hạn</dt>
              <dd>1–365 ngày</dd>
              <dd>Có thể để trống</dd>
            </div>
            <div>
              <dt>Redirect</dt>
              <dd>HTTP 302</dd>
              <dd>Không cache vĩnh viễn</dd>
            </div>
          </dl>
        </section>
      </main>

      <footer className="site-footer">
        <p>npt-shortenlink.dev · OpenAPI-first · Go + Next.js · AWS SAM</p>
      </footer>
    </div>
  );
}
