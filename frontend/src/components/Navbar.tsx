"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";
import { getAccessToken } from "@/lib/auth";
import { LinkIcon } from "./LinkIcon";
import { LogoutButton } from "./LogoutButton";

export function Navbar() {
  const pathname = usePathname();
  const [signedIn, setSignedIn] = useState(false);
  useEffect(() => {
    const updateSession = () => setSignedIn(Boolean(getAccessToken(window.sessionStorage)));
    updateSession();
    window.addEventListener("focus", updateSession);
    return () => window.removeEventListener("focus", updateSession);
  }, [pathname]);

  const items = [
    { href: "/", label: "Trang chủ" },
    ...(signedIn ? [{ href: "/dashboard/", label: "Dashboard" }] : []),
    { href: "/architecture/", label: "Kiến trúc dự án" },
  ];

  return (
    <header className="site-header">
      <div className="site-header-inner">
        <Link aria-label="Shrnk — Trang chủ" className="brand-link" href="/"><LinkIcon /><span>Shrnk</span></Link>
        <nav aria-label="Điều hướng chính" className="site-nav">
          {items.map(({ href, label }) => (
            <Link aria-current={(pathname?.replace(/\/$/, "") || "/") === (href.replace(/\/$/, "") || "/") ? "page" : undefined} className="nav-link" href={href} key={href}>{label}</Link>
          ))}
        </nav>
        <div className="site-account">{signedIn ? <LogoutButton /> : <Link aria-current={pathname?.startsWith("/login") ? "page" : undefined} className="sketch-button sketch-button-small" href="/login/">Đăng nhập</Link>}</div>
      </div>
    </header>
  );
}
