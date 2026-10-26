"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  ArrowUpRight,
  BookOpen,
  Layers2,
  Link2,
  LogIn,
  LogOut,
  Plus,
  ShieldCheck,
  Sprout,
  UserPlus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { AuthProvider, isLocalAPI, useAuth } from "@/lib/auth";

function WorkspaceShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const {
    session,
    loading,
    authError,
    clearAuthError,
    signIn,
    signUp,
    signOut,
  } = useAuth();

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
          <span className="workspace-avatar">
            {session?.email ? session.email[0].toUpperCase() : "S"}
          </span>
          <div>
            {session ? (
              <>
                <span data-testid="auth-user">
                  {session.email || session.sub}
                </span>
                <small>Authenticated owner</small>
              </>
            ) : (
              <>
                Personal workspace
                <small>
                  {loading
                    ? "Checking session…"
                    : isLocalAPI
                      ? "Local PKCE mode"
                      : "Sign in required"}
                </small>
              </>
            )}
          </div>
          <Layers2 size={15} aria-hidden="true" />
        </div>
        <div className="auth-panel" aria-label="Account session">
          {session ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="auth-button"
              onClick={() => void signOut()}
            >
              <LogOut size={15} aria-hidden="true" />
              Sign out
            </Button>
          ) : (
            <div className="auth-actions">
              <Button
                type="button"
                size="sm"
                className="auth-button"
                onClick={() => void signIn({ returnTo: pathname })}
              >
                <LogIn size={15} aria-hidden="true" />
                Sign in
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="auth-button"
                onClick={() => void signUp({ returnTo: pathname })}
              >
                <UserPlus size={15} aria-hidden="true" />
                Create account
              </Button>
            </div>
          )}
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
          <ShieldCheck size={14} aria-hidden="true" />
          {isLocalAPI
            ? "Cognito PKCE + Owner isolation"
            : "AWS Cognito PKCE + Lambda API"}
        </div>
      </aside>
      <div className="content">
        <header className="topbar">
          <span>YOUR PERSONAL LINK WORKSPACE</span>
          <span className="phase-tag">
            PHASE 03{" "}
            <span>{isLocalAPI ? "Auth & CRUD" : "Cognito + CRUD"}</span>
          </span>
        </header>
        {authError && (
          <div className="auth-alert" role="alert">
            <span>{authError}</span>
            <div className="auth-alert-actions">
              <Button
                type="button"
                size="sm"
                onClick={() => void signIn({ returnTo: pathname })}
              >
                Sign in
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={clearAuthError}
              >
                Dismiss
              </Button>
            </div>
          </div>
        )}
        <main id="main">{children}</main>
        <footer className="site-footer">
          <span>Small links. Thoughtfully made.</span>
          <span>
            {isLocalAPI ? "Shortly / Local edition" : "Shortly / Cloud edition"}
          </span>
        </footer>
      </div>
    </div>
  );
}

export function Workspace({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <WorkspaceShell>{children}</WorkspaceShell>
    </AuthProvider>
  );
}
