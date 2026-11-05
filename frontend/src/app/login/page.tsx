import Link from "next/link";
import { GoogleLoginButton } from "@/components/GoogleLoginButton";
import { Navbar } from "@/components/Navbar";

export default function LoginPage() {
  return <div className="paper-shell"><Navbar /><main className="mx-auto flex w-full max-w-6xl flex-1 items-center justify-center px-5 pb-20 pt-10 sm:px-8"><section className="sketch-card w-full max-w-md"><p className="ink-note">Cánh cửa nhỏ</p><h1 className="mt-4 font-display text-4xl font-bold">Đăng nhập để bắt đầu.</h1><p className="mt-4 leading-7 text-ink-muted">Bạn sẽ được chuyển tới Google qua Cognito để xác thực an toàn. Phiên làm việc chỉ dùng cho Shorten Link.</p><div className="mt-8"><GoogleLoginButton /></div><p className="mt-6 text-center text-xs font-semibold leading-5 text-ink-muted">Bằng cách tiếp tục, bạn đồng ý dùng Google làm nhà cung cấp đăng nhập duy nhất.</p><Link className="mt-6 inline-block text-sm font-bold underline decoration-coral decoration-2 underline-offset-4" href="/">← Về trang chủ</Link></section></main></div>;
}
