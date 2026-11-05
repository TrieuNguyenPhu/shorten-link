import type { Metadata } from "next";
import { AuthCallback } from "@/components/AuthCallback";
import { Navbar } from "@/components/Navbar";

export const metadata: Metadata = { title: "Xác thực đăng nhập" };

export default function AuthCallbackPage() {
  return <div className="paper-shell"><Navbar /><main className="mx-auto flex w-full max-w-6xl flex-1 items-center justify-center px-5 pb-20 pt-10 sm:px-8"><AuthCallback /></main></div>;
}
