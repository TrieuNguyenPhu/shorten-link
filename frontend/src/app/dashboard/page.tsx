import { LogoutButton } from "@/components/LogoutButton";
import { Navbar } from "@/components/Navbar";

export default function DashboardPlaceholder() {
  return <div className="paper-shell"><Navbar /><main className="mx-auto flex w-full max-w-6xl flex-1 items-center justify-center px-5 pb-20 pt-10 sm:px-8"><section className="sketch-card w-full max-w-2xl text-center"><p className="ink-note">Không gian của bạn</p><h1 className="mt-4 font-display text-4xl font-bold">Sẵn sàng tạo link.</h1><p className="mx-auto mt-4 max-w-lg leading-7 text-ink-muted">Phiên Google đã được xác nhận. Khu vực tạo và theo dõi link sẽ xuất hiện ở bước dashboard tiếp theo.</p><div className="mt-8 flex justify-center"><LogoutButton /></div></section></main></div>;
}
