"use client";

import { useCallback, useEffect, useState } from "react";
import NextLink from "next/link";
import { getAccessToken } from "@/lib/auth";
import { ApiError, deleteLink, listAllLinks, type Link } from "@/lib/api";
import { getPublicConfig, type PublicConfig } from "@/lib/config";
import { LinksTable } from "./LinksTable";

function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400) return "Yêu cầu chưa hợp lệ.";
    if (error.status === 401) return "Phiên đăng nhập đã hết hạn.";
    if (error.status === 404) return "Link không tồn tại hoặc không thuộc về bạn.";
    if (error.status === 429) return "API đang giới hạn tốc độ. Thử lại sau.";
    if (error.status >= 500) return "Dịch vụ đang gặp sự cố. Thử lại sau.";
  }
  return "Không tải được dữ liệu. Thử lại sau.";
}

export function LinksWorkspace({ title = "Dashboard" }: { title?: string }) {
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [links, setLinks] = useState<Link[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (currentConfig: PublicConfig, currentToken: string) => {
    setError(null);
    setLoading(true);
    try {
      setLinks(await listAllLinks(currentConfig, currentToken));
    } catch (reason) {
      if (reason instanceof ApiError && reason.status === 401) {
        window.location.replace("/login/");
        return;
      }
      setError(errorMessage(reason));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    try {
      const currentConfig = getPublicConfig();
      const currentToken = getAccessToken(window.sessionStorage);
      if (!currentToken) {
        window.location.replace("/login/");
        return;
      }
      setConfig(currentConfig);
      setToken(currentToken);
      void reload(currentConfig, currentToken);
    } catch (reason) {
      setError(errorMessage(reason));
      setLoading(false);
    }
  }, [reload]);

  async function handleDelete(code: string) {
    if (!config || !token) return;
    try {
      await deleteLink(config, token, code);
      setLinks((current) => current.filter((link) => link.code !== code));
    } catch (reason) {
      setError(errorMessage(reason));
    }
  }

  return <div className="paper-shell"><main className="mx-auto w-full max-w-6xl flex-1 px-5 pb-16 pt-10 sm:px-8 sm:pt-14"><header className="mb-8 flex flex-wrap items-end justify-between gap-5"><div><p className="ink-note">{title}</p><h1 className="mt-3 font-display text-4xl font-bold sm:text-5xl">Dữ liệu của bạn, rõ ràng hơn.</h1><p className="mt-3 max-w-2xl text-ink-muted">Theo dõi link và lượt click realtime từ cùng một chỗ.</p></div><NextLink className="sketch-button sketch-button-secondary" href="/">Về trang chủ</NextLink></header>{error ? <div aria-live="polite" className="mb-5 rounded-xl border-2 border-red-300 bg-red-50 p-4 text-sm font-bold text-red-800">{error}</div> : null}<div className="mt-8">{loading ? <section aria-live="polite" className="sketch-card text-center"><p className="font-display text-xl font-bold">Đang tải các link của bạn…</p></section> : config ? <LinksTable apiBaseUrl={config.apiBaseUrl} links={links} onDelete={handleDelete} onReconnect={() => { if (token) void reload(config, token); }} websocketUrl={config.websocketUrl} /> : null}</div></main></div>;
}
