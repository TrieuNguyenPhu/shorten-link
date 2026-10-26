"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ArrowUpRight,
  BookOpen,
  Layers2,
  Link2,
  Plus,
  Sprout,
} from "lucide-react";

export function Workspace({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  return (
    <div className="workspace">
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <aside className="sidebar">
        <Link href="/" className="brand" aria-label="Shortly home">
          <span className="brand-icon">
            <Link2 size={23} strokeWidth={2.5} />
          </span>
          shortly<span className="brand-period">.</span>
        </Link>
        <div className="workspace-label">
          <span className="workspace-avatar">S</span>
          <div>
            Personal workspace<small>Local development</small>
          </div>
          <Layers2 size={15} aria-hidden="true" />
        </div>
        <div className="nav-caption">WORKSPACE</div>
        <nav aria-label="Main navigation">
          <Link
            href="/"
            aria-current={pathname === "/" ? "page" : undefined}
            className={`nav-item ${pathname === "/" ? "selected" : ""}`}
          >
            <Plus size={18} aria-hidden="true" />
            Create a link
          </Link>
          <Link
            href="/dashboard/"
            aria-current={
              pathname.startsWith("/dashboard") ? "page" : undefined
            }
            className={`nav-item ${pathname.startsWith("/dashboard") ? "selected" : ""}`}
          >
            <BookOpen size={18} aria-hidden="true" />
            Link library
          </Link>
        </nav>
        <div className="sidebar-note">
          <Sprout size={23} aria-hidden="true" />
          <strong>
            A little less URL.
            <br />A little more possibility.
          </strong>
          <p>Make something worth sharing.</p>
          <Link href="/dashboard/">
            Explore your library <ArrowUpRight size={14} aria-hidden="true" />
          </Link>
        </div>
        <div className="local-badge">
          <span />
          Single-user local workspace
        </div>
      </aside>
      <div className="content">
        <header className="topbar">
          <span>YOUR PERSONAL LINK WORKSPACE</span>
          <span className="phase-tag">
            PHASE 01 <span>Local MVP</span>
          </span>
        </header>
        <main id="main">{children}</main>
        <footer className="site-footer">
          <span>Small links. Thoughtfully made.</span>
          <span>Shortly / Local edition</span>
        </footer>
      </div>
    </div>
  );
}
