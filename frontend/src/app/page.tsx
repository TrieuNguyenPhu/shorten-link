import Link from "next/link";
import { HomeCreateLink } from "@/components/HomeCreateLink";
import { LinkIcon } from "@/components/LinkIcon";

export default function Home() {
  return (
    <main className="mx-auto w-full max-w-6xl flex-1 px-5 pb-16 pt-12 sm:px-8 sm:pt-20" id="main-content">
      <section className="grid items-center gap-12 lg:grid-cols-[1.15fr_0.85fr]">
        <div>
          <h1 className="max-w-3xl font-display text-5xl leading-tight sm:text-7xl">Link ngắn.<br /><span className="relative inline-block text-coral">Chia sẻ dễ dàng.<span aria-hidden="true" className="scribble-underline" /></span></h1>
          <p className="mt-7 text-xl text-ink-muted">Rút gọn link, theo dõi lượt click realtime cùng Shrnk.</p>
          <div className="mt-9 flex flex-wrap gap-3">
            <a className="sketch-button" href="https://github.com/TrieuNguyenPhu/shorten-link" rel="noreferrer" target="_blank">Source Code</a>
            <Link className="sketch-button sketch-button-secondary" href="/architecture/">Kiến trúc dự án</Link>
          </div>
        </div>
        <div aria-label="Biểu tượng link Shrnk" className="relative mx-auto w-full max-w-sm rotate-[2deg]">
          <div className="sketch-card flex min-h-64 flex-col items-center justify-center gap-6 bg-sun p-8 shadow-[8px_8px_0_var(--ink)]">
            <LinkIcon className="h-28 w-28" />
            <span className="text-4xl">Dài thành ngắn.</span>
          </div>
        </div>
      </section>
      <HomeCreateLink />
    </main>
  );
}
