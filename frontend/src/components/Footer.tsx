import Link from "next/link";

export function Footer() {
  return (
    <footer className="site-footer">
      <div className="mx-auto flex w-full max-w-6xl flex-wrap items-center justify-between gap-4 px-5 py-6 sm:px-8">
        <span>Shrnk © 2026</span>
        <div className="flex flex-wrap items-center gap-5">
          <Link className="rounded-md underline-offset-4 hover:underline" href="/privacy/">
            Chính sách bảo vệ dữ liệu cá nhân
          </Link>
          <a className="rounded-md underline-offset-4 hover:underline" href="https://github.com/TrieuNguyenPhu/shorten-link" rel="noreferrer" target="_blank">
            Source Code
          </a>
        </div>
      </div>
    </footer>
  );
}
