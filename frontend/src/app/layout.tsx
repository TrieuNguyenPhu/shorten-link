import type { Metadata } from "next";
import type { ReactNode } from "react";
import "./globals.css";

export const metadata: Metadata = { title: "Shorten Link — Link ngắn, tín hiệu rõ", description: "Rút gọn link và theo dõi lượt click theo thời gian thực." };

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return <html lang="vi"><body>{children}</body></html>;
}
