import type { Metadata } from "next";
import type { ReactNode } from "react";
import localFont from "next/font/local";
import { Navbar } from "@/components/Navbar";
import { Footer } from "@/components/Footer";
import "./globals.css";

const lobster = localFont({ src: "./fonts/Lobster-Regular.ttf", variable: "--font-lobster", display: "swap", weight: "400" });

export const metadata: Metadata = {
  title: { default: "Shrnk — Link ngắn, tín hiệu rõ", template: "%s | Shrnk" },
  description: "Rút gọn link và theo dõi lượt click theo thời gian thực.",
  icons: { icon: { url: "/icon.svg", type: "image/svg+xml" } },
};

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return <html className={lobster.variable} lang="vi" suppressHydrationWarning><head><script dangerouslySetInnerHTML={{ __html: `try { var t = localStorage.getItem("theme"); document.documentElement.dataset.theme = t === "dark" || (t !== "light" && window.matchMedia("(prefers-color-scheme: dark)").matches) ? "dark" : "light"; } catch (e) { document.documentElement.dataset.theme = "light"; }` }} /></head><body><div className="paper-shell"><a className="skip-link" href="#main-content">Đến nội dung</a><Navbar />{children}<Footer /></div></body></html>;
}
