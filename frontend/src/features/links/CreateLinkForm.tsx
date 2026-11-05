"use client";

import { FormEvent, useState } from "react";
import type { PublicConfig } from "@/lib/config";
import { ApiError, createLink, type CreatedLink } from "@/lib/api";

function messageFor(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400) return "URL chưa hợp lệ. Hãy dùng địa chỉ http hoặc https.";
    if (error.status === 401) return "Phiên đăng nhập đã hết hạn. Hãy đăng nhập lại.";
    if (error.status === 429) return "Bạn thao tác hơi nhanh. Thử lại sau một chút.";
  }
  return "Chưa tạo được link. Hãy thử lại.";
}

type Props = { config: PublicConfig; token: string; onCreated: (link: CreatedLink) => void };

export function CreateLinkForm({ config, token, onCreated }: Props) {
  const [target, setTarget] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    try {
      const parsed = new URL(target.trim());
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") throw new Error("scheme");
    } catch {
      setError("URL chưa hợp lệ. Hãy dùng địa chỉ http hoặc https.");
      return;
    }
    setPending(true);
    try {
      const created = await createLink(config, token, target.trim());
      onCreated(created);
      setTarget("");
    } catch (reason) {
      setError(messageFor(reason));
    } finally {
      setPending(false);
    }
  }

  return <form className="sketch-card" onSubmit={submit}><div className="flex items-start justify-between gap-4"><div><p className="ink-note">Tạo link mới</p><h2 className="mt-2 font-display text-2xl font-bold">Một URL dài đang chờ được gọn lại.</h2></div></div><label className="mt-6 block text-sm font-bold" htmlFor="target-url">URL đích</label><div className="mt-2 flex flex-col gap-3 sm:flex-row"><input className="min-h-13 flex-1 rounded-xl border-2 border-ink bg-white/70 px-4 text-base outline-none transition focus-visible:border-coral focus-visible:ring-4 focus-visible:ring-coral/20" id="target-url" onChange={(event) => setTarget(event.target.value)} placeholder="https://example.com/article" required type="url" value={target} /><button className="sketch-button justify-center sm:min-w-36" disabled={pending} type="submit">{pending ? "Đang tạo…" : "Rút gọn"}</button></div>{error ? <p aria-live="polite" className="mt-3 text-sm font-bold text-red-700">{error}</p> : <p className="mt-3 text-xs font-semibold text-ink-muted"></p>}</form>;
}
