"use client";

import { getPublicConfig } from "@/lib/config";
import { logout } from "@/lib/auth";

export function LogoutButton() {
  return (
    <button className="sketch-button sketch-button-secondary" onClick={() => logout(getPublicConfig())} type="button">
      Đăng xuất
    </button>
  );
}
