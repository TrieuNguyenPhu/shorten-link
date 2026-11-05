"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { clearAuthSession, exchangeCodeForToken, validateCallback } from "@/lib/auth";
import { getAuthConfig } from "@/lib/config";

export function AuthCallback() {
  const [message, setMessage] = useState("Đang kiểm tra phiên đăng nhập…");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    const storage = window.sessionStorage;
    const params = new URLSearchParams(window.location.search);
    window.history.replaceState(null, "", window.location.pathname);

    async function finish() {
      try {
        if (params.get("error")) throw new Error("Google không hoàn tất đăng nhập");
        const { code } = validateCallback(storage, params);
        if (active) setMessage("Đang đổi mã đăng nhập an toàn…");
        await exchangeCodeForToken(getAuthConfig(), code, storage);
        if (active) {
          setMessage("Đăng nhập thành công. Đang mở không gian của bạn…");
          window.location.replace("/dashboard/");
        }
      } catch (reason) {
        clearAuthSession(storage);
        if (active) {
          setError(reason instanceof Error ? reason.message : "Đăng nhập chưa hoàn tất");
          setMessage("");
        }
      }
    }

    void finish();
    return () => {
      active = false;
    };
  }, []);

  return (
    <section aria-live="polite" className="sketch-card mx-auto w-full max-w-xl text-center">
      <p className="ink-note">OAuth callback</p>
      <h1 className="mt-3 font-display text-3xl font-bold">{error ? "Chưa thể đăng nhập" : "Một nét nữa thôi"}</h1>
      {message ? <p className="mt-4 text-ink-muted">{message}</p> : null}
      {error ? (
        <>
          <p className="mt-4 rounded-xl border-2 border-red-300 bg-red-50 p-3 text-left text-sm font-semibold text-red-800">{error}</p>
          <Link className="sketch-button mt-6 inline-flex" href="/login/">Quay lại đăng nhập</Link>
        </>
      ) : <div aria-hidden="true" className="mx-auto mt-7 h-2 w-40 overflow-hidden rounded-full bg-paper-deep"><div className="h-full w-1/2 animate-pulse rounded-full bg-sun" /></div>}
    </section>
  );
}
