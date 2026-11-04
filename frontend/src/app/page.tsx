import Link from "next/link";
import { Navbar } from "@/components/Navbar";

const features = [
  ["01", "Một đường link gọn", "Biến URL dài thành một điểm đến dễ nhớ, dễ chia sẻ."],
  ["02", "Số liệu thật", "Theo dõi lượt click sau khi hệ thống đã ghi nhận sự kiện."],
  ["03", "Cập nhật tức thì", "Khi mở dashboard, nhịp click mới sẽ hiện theo thời gian thực."],
];

export default function Home() {
  return (
    <div className="paper-shell">
      <Navbar />
      <main className="mx-auto w-full max-w-6xl flex-1 px-5 pb-16 pt-10 sm:px-8 sm:pt-20">
        <section className="grid items-center gap-12 lg:grid-cols-[1.05fr_0.95fr]">
          <div>
            <p className="ink-note">Link ngắn, tín hiệu rõ</p>
            <h1 className="mt-5 max-w-3xl font-display text-5xl font-bold leading-[0.98] tracking-[-0.04em] text-ink sm:text-7xl">
              Chia sẻ điều quan trọng, <span className="relative inline-block text-coral">gọn hơn<span aria-hidden="true" className="scribble-underline" /></span>.
            </h1>
            <p className="mt-7 max-w-xl text-lg leading-8 text-ink-muted sm:text-xl">Shorten Link giúp bạn rút gọn link trong vài giây và nhìn thấy lượt click thật đang chuyển động.</p>
            <div className="mt-9 flex flex-col gap-3 sm:flex-row"><Link className="sketch-button" href="/login/">Bắt đầu với Google <span aria-hidden="true">→</span></Link><a className="sketch-button sketch-button-secondary" href="#how-it-works">Xem cách hoạt động</a></div>
            <p className="mt-4 text-sm font-semibold text-ink-muted">Đăng nhập qua Google và Cognito. Không có mật khẩu riêng.</p>
          </div>
          <div aria-label="Sơ đồ luồng link ngắn và lượt click" className="relative mx-auto w-full max-w-md rotate-[2deg]">
            <div className="sketch-card bg-sun p-7 shadow-[8px_8px_0_var(--ink)]"><div className="flex items-center justify-between text-xs font-bold uppercase tracking-[0.22em]"><span>Link note</span><span aria-hidden="true">✦</span></div><div className="mt-8 space-y-3 font-display text-2xl font-bold"><p className="rounded-xl border-2 border-dashed border-ink/60 bg-paper/60 px-4 py-3">URL dài</p><p aria-hidden="true" className="text-center text-coral">↓</p><p className="rounded-xl border-2 border-ink bg-white/70 px-4 py-3">Link ngắn</p><p aria-hidden="true" className="text-center text-coral">↓</p><p className="rounded-xl border-2 border-dashed border-ink/60 bg-paper/60 px-4 py-3">Click realtime</p></div></div>
            <p className="mt-4 -rotate-[5deg] text-center font-display text-lg font-bold text-coral">↳ Theo dõi bằng dữ liệu thật.</p>
          </div>
        </section>
        <section className="mt-24" id="how-it-works"><div className="mb-8 flex items-end justify-between gap-5"><div><p className="ink-note">Bản đồ nhỏ</p><h2 className="mt-3 font-display text-3xl font-bold sm:text-4xl">Ít bước, nhiều rõ ràng.</h2></div><span aria-hidden="true" className="hidden text-4xl text-coral sm:block">✎</span></div><div className="grid gap-5 md:grid-cols-3">{features.map(([number, title, copy]) => <article className="sketch-card" key={number}><p className="font-display text-2xl font-bold text-coral">{number}</p><h3 className="mt-8 font-display text-xl font-bold">{title}</h3><p className="mt-3 leading-7 text-ink-muted">{copy}</p></article>)}</div></section>
      </main>
      <footer className="mx-auto flex w-full max-w-6xl justify-between px-5 pb-8 text-sm font-semibold text-ink-muted sm:px-8"><span>Shorten Link / 2026</span><span className="font-display">Keep it short. Keep it useful.</span></footer>
    </div>
  );
}
