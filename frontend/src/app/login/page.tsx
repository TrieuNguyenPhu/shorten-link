import type { Metadata } from "next";
import { GoogleLoginButton } from "@/components/GoogleLoginButton";
import { LinkIcon } from "@/components/LinkIcon";

export const metadata: Metadata = { title: "Đăng nhập" };

export default function LoginPage() {
  return (
    <main className="mx-auto flex w-full max-w-6xl flex-1 items-center justify-center px-5 py-16 sm:px-8" id="main-content">
      <section className="sketch-card w-full max-w-md px-6 py-10 text-center sm:px-10">
        <LinkIcon className="mx-auto h-16 w-16" />
        <h1 className="mt-6 font-display text-4xl">Đăng nhập Shrnk</h1>
        <div className="mt-8"><GoogleLoginButton /></div>
      </section>
    </main>
  );
}
