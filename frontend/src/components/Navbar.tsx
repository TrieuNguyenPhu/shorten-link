import Link from "next/link";

export function Navbar() {
  return (
    <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
      <Link className="flex items-center gap-3 rounded-md focus-visible:outline-3 focus-visible:outline-offset-4 focus-visible:outline-ink" href="/">
        <span aria-hidden="true" className="grid h-10 w-10 rotate-[-5deg] place-items-center rounded-[45%] border-2 border-ink bg-sun text-xl shadow-[3px_3px_0_var(--ink)]">↗</span>
        <span className="font-display text-lg font-bold tracking-tight">Shorten Link</span>
      </Link>
      <nav aria-label="Điều hướng chính" className="flex items-center gap-2 text-sm font-semibold sm:gap-5">
        <Link className="rounded-md px-2 py-2 text-ink-muted underline decoration-transparent decoration-2 underline-offset-4 transition hover:text-ink hover:decoration-ink focus-visible:outline-3 focus-visible:outline-offset-2 focus-visible:outline-ink" href="/">Trang chủ</Link>
        <Link className="sketch-button sketch-button-small" href="/login/">Đăng nhập</Link>
      </nav>
    </header>
  );
}
