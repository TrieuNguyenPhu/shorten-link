"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { clearAuthSession, exchangeCodeForToken, validateCallback } from "@/lib/auth";
import { getAuthConfig } from "@/lib/config";

export function AuthCallback() {
  const [message, setMessage] = useState("Đang đăng nhập…");
  const [error, setError] = useState<string | null>(null);
  const completion = useRef<Promise<void> | null>(null);

  useEffect(() => {
    let active = true;
    if (!completion.current) {
      const storage = window.sessionStorage;
      const params = new URLSearchParams(window.location.search);
      window.history.replaceState(null, "", window.location.pathname);
      completion.current = (async () => {
        try {
          if (params.get("error")) throw new Error("Google không hoàn tất đăng nhập");
          const { code } = validateCallback(storage, params);
          setMessage("Đang đăng nhập…");
          await exchangeCodeForToken(getAuthConfig(), code, storage);
        } catch (reason) {
          clearAuthSession(storage);
          throw reason;
        }
      })();
    }

    void completion.current.then(() => {
      if (active) {
        setMessage("Đang về trang chủ…");
        window.location.replace("/");
      }
    }, (reason) => {
      if (active) {
        setError(reason instanceof Error ? reason.message : "Đăng nhập chưa hoàn tất");
        setMessage("");
      }
    });
    return () => {
      active = false;
    };
  }, []);

  return (
    <section aria-live="polite" className="sketch-card mx-auto w-full max-w-xl text-center">
      <h1 className="mt-3 font-display text-3xl font-bold">{error ? "Chưa thể đăng nhập" : "Đăng nhập Shrnk"}</h1>
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
