"use client";

import { useState } from "react";
import { getAuthConfig } from "@/lib/config";
import { startGoogleLogin } from "@/lib/auth";

export function GoogleLoginButton() {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    setPending(true);
    setError(null);
    try {
      await startGoogleLogin(getAuthConfig());
    } catch {
      setPending(false);
      setError("Chưa thể mở đăng nhập Google. Kiểm tra cấu hình môi trường rồi thử lại.");
    }
  }

  return (
    <div className="space-y-3">
      <button className="sketch-button w-full justify-center" disabled={pending} onClick={handleClick} type="button">
        <span aria-hidden="true" className="grid h-6 w-6 place-items-center rounded-full bg-white text-sm font-bold text-[#4285f4]">G</span>
        {pending ? "Đang mở Google…" : "Tiếp tục với Google"}
      </button>
      {error ? <p aria-live="polite" className="text-sm font-semibold text-red-700">{error}</p> : null}
    </div>
  );
}
