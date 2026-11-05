import type { Metadata } from "next";
import type { ReactNode } from "react";
import "./globals.css";

export const metadata: Metadata = {
  title: { default: "Shrnk — Link ngắn, tín hiệu rõ", template: "%s | Shrnk" },
  description: "Rút gọn link và theo dõi lượt click theo thời gian thực.",
  icons: { icon: { url: "/icon.svg", type: "image/svg+xml" } },
};

export default function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  return <html lang="vi"><body>{children}</body></html>;
}
