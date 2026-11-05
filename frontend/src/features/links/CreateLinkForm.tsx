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

type Props = { config: PublicConfig; token: string };

export function CreateLinkForm({ config, token }: Props) {
  const [created, setCreated] = useState<CreatedLink | null>(null);
  const [copied, setCopied] = useState(false);
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
      setCreated(created);
      setCopied(false);
      setTarget("");
    } catch (reason) {
      setError(messageFor(reason));
    } finally {
      setPending(false);
    }
  }

  async function copyResult() {
    if (!created) return;
    try {
      await window.navigator.clipboard.writeText(created.shortUrl);
      setCopied(true);
    } catch {
      setError("Chưa sao chép được. Hãy chọn link để sao chép.");
    }
  }

  return (
    <form className="sketch-card" onSubmit={submit}>
      <h2 className="font-display text-3xl">Tạo link mới</h2>
      <label className="mt-6 block" htmlFor="target-url">URL đích</label>
      <div className="mt-2 flex flex-col gap-3 sm:flex-row">
        <input className="min-h-13 min-w-0 flex-1 rounded-xl border-2 border-ink bg-white/70 px-4 text-base outline-none transition focus-visible:border-coral focus-visible:ring-4 focus-visible:ring-coral/20" id="target-url" onChange={(event) => setTarget(event.target.value)} placeholder="https://example.com/article" required type="url" value={target} />
        <button className="sketch-button justify-center sm:min-w-36" disabled={pending} type="submit">{pending ? "Đang tạo…" : "Rút gọn"}</button>
      </div>
      {error ? <p aria-live="polite" className="mt-3 text-red-700">{error}</p> : null}
      {created ? (
        <div aria-live="polite" className="mt-5 flex flex-wrap items-center gap-3 rounded-xl border-2 border-dashed border-ink/40 bg-paper p-4">
          <a aria-label={`Link mới: ${created.shortUrl}`} className="min-w-0 flex-1 break-all underline decoration-coral underline-offset-4" href={created.shortUrl} rel="noreferrer" target="_blank">{created.shortUrl}</a>
          <button className="sketch-button sketch-button-secondary sketch-button-small" onClick={() => void copyResult()} type="button">{copied ? "Đã copy" : "Copy"}</button>
        </div>
      ) : null}
    </form>
  );
}
