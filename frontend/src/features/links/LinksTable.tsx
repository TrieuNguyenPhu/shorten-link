"use client";

import { useEffect, useState } from "react";
import type { Link } from "@/lib/api";
import { useRealtimeClicks, type RealtimeStatus } from "@/hooks/useRealtimeClicks";

function statusLabel(status: RealtimeStatus): string {
  if (status === "open") return "realtime";
  if (status === "connecting" || status === "reconnecting") return "đang nối";
  if (status === "closed") return "đã đóng";
  return "chờ nối";
}

function ClickCell({ link, websocketUrl, onReconnect }: { link: Link; websocketUrl: string; onReconnect: () => void }) {
  const [baseClicks, setBaseClicks] = useState(link.clicks);
  const { clicks: pushedClicks, status } = useRealtimeClicks({ code: link.code, websocketUrl, onReconnect });
  useEffect(() => setBaseClicks((current) => Math.max(current, link.clicks)), [link.clicks]);
  const clicks = Math.max(baseClicks, pushedClicks);
  return <div className="flex items-center gap-2"><strong className="font-display text-lg">{clicks}</strong><span className="rounded-full border border-ink/30 px-2 py-0.5 text-[10px] font-black uppercase tracking-wide text-ink-muted">{statusLabel(status)}</span></div>;
}

type Props = { links: Link[]; apiBaseUrl: string; websocketUrl: string; onDelete: (code: string) => Promise<void>; onReconnect: () => void };

export function LinksTable({ links, apiBaseUrl, websocketUrl, onDelete, onReconnect }: Props) {
  const [copyState, setCopyState] = useState<string | null>(null);
  if (links.length === 0) return <section className="sketch-card text-center"><p className="ink-note">Dashboard</p><h2 className="mt-3 font-display text-2xl font-bold">Chưa có link nào.</h2><p className="mt-3 text-ink-muted">Tạo link đầu tiên ở phía trên để bắt đầu thấy dữ liệu thật.</p></section>;

  async function copy(code: string) {
    const url = `${apiBaseUrl}/${code}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopyState(code);
      window.setTimeout(() => setCopyState((current) => current === code ? null : current), 1600);
    } catch {
      setCopyState(null);
    }
  }

  return <section className="sketch-card overflow-hidden p-0"><div className="flex items-center justify-between gap-4 border-b-2 border-dashed border-ink/30 px-5 py-5 sm:px-6"><div><p className="ink-note">Dashboard</p><h2 className="mt-2 font-display text-2xl font-bold">Những link của bạn</h2></div></div><div className="overflow-x-auto"><table className="w-full min-w-[680px] text-left text-sm"><thead className="bg-paper-deep/50 text-xs uppercase tracking-[0.14em] text-ink-muted"><tr><th className="px-5 py-4 font-black sm:px-6">Link</th><th className="px-5 py-4 font-black">Clicks</th><th className="px-5 py-4 font-black">Đích đến</th><th className="px-5 py-4 font-black">Thao tác</th></tr></thead><tbody>{links.map((link) => <tr className="border-t border-ink/10" key={link.code}><td className="px-5 py-4 sm:px-6"><a className="font-display font-bold text-coral underline decoration-coral/30 underline-offset-4" href={`${apiBaseUrl}/${link.code}`} rel="noreferrer" target="_blank">/{link.code}</a></td><td className="px-5 py-4"><ClickCell link={link} onReconnect={onReconnect} websocketUrl={websocketUrl} /></td><td className="max-w-xs truncate px-5 py-4 text-ink-muted" title={link.target}>{link.target}</td><td className="px-5 py-4"><div className="flex flex-wrap gap-2"><button className="rounded-lg border-2 border-ink px-2.5 py-1.5 text-xs font-bold transition hover:bg-sun focus-visible:outline-3 focus-visible:outline-coral" onClick={() => void copy(link.code)} type="button">{copyState === link.code ? "Đã copy" : "Copy"}</button><button className="rounded-lg border-2 border-red-300 px-2.5 py-1.5 text-xs font-bold text-red-700 transition hover:bg-red-50 focus-visible:outline-3 focus-visible:outline-coral" onClick={() => void onDelete(link.code)} type="button">Xóa</button></div></td></tr>)}</tbody></table></div></section>;
}
