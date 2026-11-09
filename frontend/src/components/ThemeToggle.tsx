"use client";

import { Moon, Sun } from "lucide-react";
import { useEffect, useState } from "react";

export function ThemeToggle() {
  const [dark, setDark] = useState(false);
  useEffect(() => {
    setDark(document.documentElement.dataset.theme === "dark");
  }, []);

  function toggle() {
    const next = document.documentElement.dataset.theme !== "dark";
    document.documentElement.dataset.theme = next ? "dark" : "light";
    try { localStorage.setItem("theme", next ? "dark" : "light"); } catch { /* Storage may be unavailable. */ }
    setDark(next);
  }

  return <button aria-label={dark ? "Chuyển sang chế độ sáng" : "Chuyển sang chế độ tối"} className="theme-toggle" onClick={toggle} type="button">{dark ? <Sun size={20} aria-hidden="true" /> : <Moon size={20} aria-hidden="true" />}</button>;
}
