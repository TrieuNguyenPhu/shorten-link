"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { getAccessToken } from "@/lib/auth";
import { LinkIcon } from "./LinkIcon";
import { LogoutButton } from "./LogoutButton";
export function Navbar() { const [signedIn, setSignedIn] = useState(false); useEffect(() => setSignedIn(Boolean(getAccessToken(window.sessionStorage))), []); return <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-5 py-5 sm:px-8"><Link className="flex items-center gap-3 rounded-md" href="/"><LinkIcon /><span className="font-display text-lg font-bold tracking-tight">Shrnk</span></Link><nav aria-label="Điều hướng chính" className="flex items-center gap-2 text-sm font-semibold sm:gap-5"><Link className="rounded-md px-2 py-2 text-ink-muted" href="/">Trang chủ</Link>{signedIn ? <><Link className="sketch-button sketch-button-small" href="/dashboard/">Dashboard</Link><span className="sr-only">Đăng xuất</span><LogoutButton /></> : <Link className="sketch-button sketch-button-small" href="/login/">Đăng nhập</Link>}</nav></header>; }
