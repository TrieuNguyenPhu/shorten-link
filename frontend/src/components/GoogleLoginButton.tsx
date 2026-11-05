"use client";

import Link from "next/link";
import { useState } from "react";
import { getAuthConfig } from "@/lib/config";
import { startGoogleLogin } from "@/lib/auth";

export function GoogleLoginButton() {
  const [consented, setConsented] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    if (!consented) {
      setError("Vui lòng đọc và đồng ý với Chính sách bảo vệ dữ liệu cá nhân trước khi đăng nhập.");
      return;
    }
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
    <div className="space-y-4">
      <label className="flex cursor-pointer items-start gap-3 rounded-xl border-2 border-dashed border-ink/40 bg-paper p-3.5 text-left text-sm leading-relaxed text-ink">
        <input
          checked={consented}
          className="mt-1 h-5 w-5 shrink-0 accent-coral"
          onChange={(event) => {
            setConsented(event.target.checked);
            if (event.target.checked) setError(null);
          }}
          type="checkbox"
        />
        <span>
          Tôi đồng ý để Shrnk thu thập và xử lý địa chỉ email, mã định danh tài khoản Google nhằm xác thực đăng nhập và quản lý liên kết rút gọn (lưu trữ trên hạ tầng AWS tại Singapore) theo{" "}
          <Link className="font-bold text-coral underline underline-offset-4" href="/privacy/">
            Chính sách bảo vệ dữ liệu cá nhân
          </Link>
          .
        </span>
      </label>
      <button className="sketch-button w-full justify-center" disabled={pending} onClick={handleClick} type="button">
        <span aria-hidden="true" className="grid h-6 w-6 place-items-center rounded-full bg-white text-sm font-bold text-[#4285f4]">G</span>
        {pending ? "Đang mở Google…" : "Tiếp tục với Google"}
      </button>
      {error ? <p aria-live="polite" className="text-sm font-semibold text-red-700">{error}</p> : null}
    </div>
  );
}
