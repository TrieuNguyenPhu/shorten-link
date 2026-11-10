"use client";

import { useEffect, useState } from "react";
import NextLink from "next/link";
import type { Link } from "@/lib/api";
import { useRealtimeClicks, type RealtimeStatus } from "@/hooks/useRealtimeClicks";

function statusLabel(status: RealtimeStatus): string {
  if (status === "open") return "realtime";
  if (status === "connecting" || status === "reconnecting") return "đang nối";
  if (status === "closed") return "đã đóng";
  return "chờ nối";
}

function expiryLabel(expiresAt?: number): string {
  if (!expiresAt) return "Vĩnh viễn";
  return `Hết hạn ${new Intl.DateTimeFormat("vi-VN", { dateStyle: "medium" }).format(new Date(expiresAt * 1000))}`;
}

function ClickCell({ link, websocketUrl, onReconnect }: { link: Link; websocketUrl: string; onReconnect: () => void }) {
  const [baseClicks, setBaseClicks] = useState(link.clicks);
  const { clicks: pushedClicks, status } = useRealtimeClicks({ code: link.code, realtimeTicket: link.realtimeTicket, websocketUrl, onReconnect });
  useEffect(() => setBaseClicks((current) => Math.max(current, link.clicks)), [link.clicks]);
  const clicks = Math.max(baseClicks, pushedClicks);
  return <div className="flex items-center gap-2"><strong className="font-display text-lg">{clicks}</strong><span className="rounded-full border border-ink/30 px-2 py-0.5 text-sm text-ink-muted">{statusLabel(status)}</span></div>;
}

type Props = { links: Link[]; apiBaseUrl: string; websocketUrl: string; onDelete: (code: string) => Promise<void>; onReconnect: () => void };

export function LinksTable({ links, apiBaseUrl, websocketUrl, onDelete, onReconnect }: Props) {
  const [copyState, setCopyState] = useState<string | null>(null);
  const [copyError, setCopyError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<Set<string>>(new Set());
  if (links.length === 0) return <section className="sketch-card text-center"><h2 className="font-display text-2xl">Chưa có link nào</h2><NextLink className="sketch-button mt-6" href="/#create-link">Tạo link mới</NextLink></section>;

  async function copy(link: Link) {
    const url = link.shortUrl || `${apiBaseUrl}/${link.code}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopyState(link.code);
      setCopyError(null);
      window.setTimeout(() => setCopyState((current) => current === link.code ? null : current), 1600);
    } catch {
      setCopyError(link.code);
      setCopyState(null);
    }
  }

  async function remove(code: string) {
    if (deleting.has(code)) return;
    setDeleting((current) => new Set(current).add(code));
    try {
      await onDelete(code);
    } finally {
      setDeleting((current) => {
        const next = new Set(current);
        next.delete(code);
        return next;
      });
    }
  }

  return <section className="sketch-card overflow-hidden p-0"><div className="overflow-x-auto"><table aria-label="Các link của bạn" className="w-full min-w-[820px] text-left text-sm"><thead className="bg-paper-deep/50 text-base text-ink-muted"><tr><th className="px-5 py-4 font-black sm:px-6">Link</th><th className="px-5 py-4 font-black">Clicks</th><th className="px-5 py-4 font-black">Đích đến</th><th className="px-5 py-4 font-black">Thời hạn</th><th className="px-5 py-4 font-black">Thao tác</th></tr></thead><tbody>{links.map((link) => <tr className="border-t border-ink/10" key={link.code}><td className="px-5 py-4 sm:px-6"><a className="font-display font-bold text-coral underline decoration-coral/30 underline-offset-4" href={link.shortUrl || `${apiBaseUrl}/${link.code}`} rel="noreferrer" target="_blank">/{link.code}</a></td><td className="px-5 py-4"><ClickCell link={link} onReconnect={onReconnect} websocketUrl={websocketUrl} /></td><td className="max-w-xs break-all px-5 py-4 text-ink-muted" title={link.target}><a href={link.target} rel="noreferrer" target="_blank">{link.target}</a></td><td className="whitespace-nowrap px-5 py-4 text-ink-muted">{expiryLabel(link.expiresAt)}</td><td className="px-5 py-4"><div className="flex flex-wrap gap-2"><button className="rounded-lg border-2 border-ink px-2.5 py-1.5 text-sm font-bold transition hover:bg-sun focus-visible:outline-3 focus-visible:outline-coral" onClick={() => void copy(link)} type="button">{copyState === link.code ? "Đã copy" : "Copy"}</button><button aria-busy={deleting.has(link.code)} className="rounded-lg border-2 border-red-300 px-2.5 py-1.5 text-sm font-bold text-red-700 transition hover:bg-red-50 focus-visible:outline-3 focus-visible:outline-coral disabled:cursor-wait disabled:opacity-60" disabled={deleting.has(link.code)} onClick={() => void remove(link.code)} type="button">{deleting.has(link.code) ? "Đang xóa…" : "Xóa"}</button></div></td></tr>)}</tbody></table></div>{copyError ? <p aria-live="polite" className="border-t border-red-200 px-5 py-3 text-sm text-red-700">Không sao chép được. Hãy chọn URL để sao chép.</p> : null}</section>;
}
