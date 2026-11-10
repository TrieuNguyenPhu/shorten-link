"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getAccessToken } from "@/lib/auth";
import { ApiError, deleteLink, listAllLinks, type CreatedLink, type Link } from "@/lib/api";
import { getPublicConfig, type PublicConfig } from "@/lib/config";
import { CreateLinkForm } from "./CreateLinkForm";
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

export function mergeLinkSnapshot(current: Link[], snapshot: Link[], pendingCreated: Set<string>, deletedCodes: Set<string>): { links: Link[]; pendingCreated: Set<string> } {
  const nextPending = new Set(pendingCreated);
  const visibleSnapshot = snapshot.filter((link) => !deletedCodes.has(link.code));
  const snapshotCodes = new Set(visibleSnapshot.map((link) => link.code));
  for (const code of nextPending) {
    if (snapshotCodes.has(code)) nextPending.delete(code);
  }
  const preserved = current.filter((link) => nextPending.has(link.code) && !deletedCodes.has(link.code) && !snapshotCodes.has(link.code));
  const byCode = new Map<string, Link>();
  for (const link of [...preserved, ...visibleSnapshot]) byCode.set(link.code, link);
  return { links: [...byCode.values()], pendingCreated: nextPending };
}

export function LinksWorkspace() {
  const [config, setConfig] = useState<PublicConfig | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [links, setLinks] = useState<Link[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useRef<Promise<void> | null>(null);
  const deletedCodes = useRef(new Set<string>());
  const pendingCreated = useRef(new Set<string>());
  const reload = useCallback((currentConfig: PublicConfig) => {
    if (refresh.current) return refresh.current;
    setError(null);
    // Background snapshots must preserve the table and its live subscriptions.
    refresh.current = (async () => {
      try {
        const freshToken = getAccessToken(window.sessionStorage);
        if (!freshToken) {
          window.location.replace(`/login/?returnTo=${encodeURIComponent(window.location.pathname)}`);
          return;
        }
        const snapshot = await listAllLinks(currentConfig, freshToken);
        setLinks((current) => {
          const merged = mergeLinkSnapshot(current, snapshot, pendingCreated.current, deletedCodes.current);
          pendingCreated.current = merged.pendingCreated;
          return merged.links;
        });
      } catch (reason) {
        if (reason instanceof ApiError && reason.status === 401) {
          window.location.replace(`/login/?returnTo=${encodeURIComponent(window.location.pathname)}`);
          return;
        }
        setError(errorMessage(reason));
      } finally {
        setLoading(false);
        refresh.current = null;
      }
    })();
    return refresh.current;
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
      void reload(currentConfig);
    } catch (reason) {
      setError(errorMessage(reason));
      setLoading(false);
    }
  }, [reload]);

  async function handleDelete(code: string) {
    if (!config || !token) return;
    try {
      const freshToken = getAccessToken(window.sessionStorage);
      if (!freshToken) {
        window.location.replace(`/login/?returnTo=${encodeURIComponent(window.location.pathname)}`);
        return;
      }
      await deleteLink(config, freshToken, code);
      deletedCodes.current.add(code);
      setLinks((current) => current.filter((link) => link.code !== code));
    } catch (reason) {
      if (reason instanceof ApiError && reason.status === 401) {
        window.location.replace(`/login/?returnTo=${encodeURIComponent(window.location.pathname)}`);
        return;
      }
      setError(errorMessage(reason));
    }
  }

  function handleCreated(created: CreatedLink) {
    deletedCodes.current.delete(created.code);
    pendingCreated.current.add(created.code);
    setLinks((current) => [
      {
        code: created.code,
        target: created.target,
        shortUrl: created.shortUrl,
        expiresAt: created.expiresAt,
        clicks: 0,
        createdAt: new Date().toISOString(),
      },
      ...current.filter((link) => link.code !== created.code),
    ]);
  }

  return (
    <main className="mx-auto w-full max-w-6xl flex-1 px-5 pb-16 pt-10 sm:px-8 sm:pt-14" id="main-content">
      <h1 className="mb-8 font-display text-4xl sm:text-5xl">Dashboard</h1>
      {config && token ? (
        <div className="mb-8">
          <CreateLinkForm config={config} onCreated={handleCreated} token={token} />
        </div>
      ) : null}
      {error ? <div aria-live="polite" className="mb-5 rounded-xl border-2 border-red-300 bg-red-50 p-4 text-red-800">{error}</div> : null}
      {loading ? <section aria-live="polite" className="sketch-card text-center">Đang tải…</section> : config ? (
        <LinksTable apiBaseUrl={config.apiBaseUrl} links={links} onDelete={handleDelete} onReconnect={() => { if (token) void reload(config); }} websocketUrl={config.websocketUrl} />
      ) : null}
    </main>
  );
}
