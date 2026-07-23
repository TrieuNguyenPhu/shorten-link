import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "NPT ShortenLink — Frontend preview",
  description:
    "Bản xem trước frontend biệt lập cho ứng dụng rút gọn URL hiện tại.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="vi">
      <body>{children}</body>
    </html>
  );
}
