"use client";

import { getAuthConfig } from "@/lib/config";
import { logout } from "@/lib/auth";

export function LogoutButton() {
  return (
    <button className="sketch-button sketch-button-secondary sketch-button-small" onClick={() => logout(getAuthConfig())} type="button">
      Đăng xuất
    </button>
  );
}
