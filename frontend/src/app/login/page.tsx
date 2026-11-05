import type { Metadata } from "next";
import Link from "next/link";
import { GoogleLoginButton } from "@/components/GoogleLoginButton";
import { Navbar } from "@/components/Navbar";

export const metadata: Metadata = { title: "Đăng nhập" };

export default function LoginPage() {
  return <div className="paper-shell"><Navbar /><main className="mx-auto flex w-full max-w-6xl flex-1 items-center justify-center px-5 pb-20 pt-10 sm:px-8"><section className="sketch-card w-full max-w-md"><p className="ink-note">Cánh cửa nhỏ</p><h1 className="mt-4 font-display text-4xl font-bold">Đăng nhập để bắt đầu.</h1><div className="mt-8"><GoogleLoginButton /></div><Link className="mt-6 inline-block text-sm font-bold underline decoration-coral decoration-2 underline-offset-4" href="/">← Về trang chủ</Link></section></main></div>;
}
