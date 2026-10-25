"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  ArrowUpRight,
  Link2,
  LoaderCircle,
  Plus,
  Power,
  RefreshCw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/copy-button";
import { api, shortURL, type ShortLink } from "@/lib/api";

function status(link: ShortLink, now: number) {
  if (!link.enabled) return "Disabled";
  return link.expiresAt && new Date(link.expiresAt).getTime() <= now
    ? "Expired"
    : "Active";
}

function date(value: string) {
  return (
    new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
      timeZone: "UTC",
    }).format(new Date(value)) + " UTC"
  );
}

export function LinkLibrary() {
  const [links, setLinks] = useState<ShortLink[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [pendingCode, setPendingCode] = useState("");
  const [notice, setNotice] = useState("");
  const [now, setNow] = useState(0);
  const latestLoad = useRef<symbol | null>(null);

  const load = useCallback((nextCursor?: string) => {
    const request = Symbol();
    latestLoad.current = request;
    return api
      .list(nextCursor)
      .then((page) => {
        if (request !== latestLoad.current) return;
        setLinks((previous) =>
          nextCursor
            ? [
                ...previous,
                ...page.links.filter(
                  (link) => !previous.some((item) => item.code === link.code),
                ),
              ]
            : page.links,
        );
        setCursor(page.nextCursor);
        setNow(Date.now());
      })
      .catch((err: unknown) => {
        if (request !== latestLoad.current) return;
        setError(err instanceof Error ? err.message : "Unable to load links.");
      })
      .finally(() => {
        if (request === latestLoad.current) setLoading(false);
      });
  }, []);

  function refresh(nextCursor?: string) {
    setLoading(true);
    setError("");
    void load(nextCursor);
  }

  useEffect(() => {
    void load();
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => {
      latestLoad.current = null;
      clearInterval(timer);
    };
  }, [load]);

  async function toggle(link: ShortLink) {
    setPendingCode(link.code);
    setError("");
    setNotice("");
    try {
      const updated = await api.setEnabled(link.code, !link.enabled);
      setLinks((previous) =>
        previous.map((item) => (item.code === updated.code ? updated : item)),
      );
      setNotice(`${updated.code} ${updated.enabled ? "enabled" : "disabled"}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to update link.");
    } finally {
      setPendingCode("");
    }
  }

  return (
    <>
      <div className="page-heading">
        <div>
          <p className="eyebrow">YOUR COLLECTION</p>
          <h1>Your links, all together.</h1>
          <p>A home for every destination you’ve made easier to share.</p>
        </div>
        <Button asChild>
          <Link href="/">
            <Plus aria-hidden="true" />
            Create a link
          </Link>
        </Button>
      </div>
      <section className="library-panel" aria-labelledby="library-heading">
        <div className="library-toolbar">
          <h2 id="library-heading">
            Link library <span>{links.length} loaded</span>
          </h2>
          <Button
            variant="ghost"
            size="sm"
            disabled={loading || !!pendingCode}
            onClick={() => refresh()}
            aria-label="Refresh links"
          >
            <RefreshCw
              className={loading ? "animate-spin" : ""}
              aria-hidden="true"
            />
            Refresh
          </Button>
        </div>
        {error && (
          <div className="error-banner" role="alert">
            <span>{error}</span>
            <Button
              variant="outline"
              size="sm"
              disabled={loading}
              onClick={() => refresh()}
            >
              Try again
            </Button>
          </div>
        )}
        <div className="sr-only" role="status">
          {notice}
        </div>
        {loading && (
          <p className="loading-state" role="status">
            <LoaderCircle
              className="animate-spin"
              size={18}
              aria-hidden="true"
            />
            Loading links…
          </p>
        )}
        {!loading && !error && links.length === 0 && (
          <div className="empty-state">
            <span>
              <Link2 size={27} aria-hidden="true" />
            </span>
            <h3>Your next good link starts here</h3>
            <p>
              Turn a long address into something simple.
              <br />
              Your links will be waiting here when you do.
            </p>
            <Button asChild>
              <Link href="/">
                Create your first link <Plus aria-hidden="true" />
              </Link>
            </Button>
          </div>
        )}
        <div className="link-list">
          {links.map((link) => {
            const currentStatus = status(link, now);
            return (
              <article
                className="link-row"
                key={link.code}
                aria-label={`Link ${link.code}`}
              >
                <div className="link-row-icon">
                  <Link2 size={19} aria-hidden="true" />
                </div>
                <div className="link-details">
                  <div className="link-title">
                    <a
                      href={shortURL(link.code)}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {link.code}
                      <ArrowUpRight size={14} aria-hidden="true" />
                    </a>
                    <span
                      className={`status-pill ${currentStatus.toLowerCase()}`}
                    >
                      {currentStatus}
                    </span>
                  </div>
                  <p className="destination" title={link.url}>
                    {link.url}
                  </p>
                  <div className="link-meta">
                    <span>Created {date(link.createdAt)}</span>
                    <span>
                      {link.expiresAt
                        ? `Expires ${date(link.expiresAt)}`
                        : "No expiration"}
                    </span>
                  </div>
                </div>
                <div className="row-actions">
                  <CopyButton value={shortURL(link.code)} />
                  <Button
                    variant="ghost"
                    size="sm"
                    disabled={!!pendingCode || loading}
                    onClick={() => void toggle(link)}
                    aria-label={link.enabled ? "Disable link" : "Enable link"}
                  >
                    {pendingCode === link.code ? (
                      <LoaderCircle
                        className="animate-spin"
                        aria-hidden="true"
                      />
                    ) : (
                      <Power aria-hidden="true" />
                    )}
                    {link.enabled ? "Disable" : "Enable"}
                  </Button>
                </div>
              </article>
            );
          })}
        </div>
        {cursor && (
          <div className="load-more">
            <Button
              variant="outline"
              disabled={loading || !!pendingCode}
              onClick={() => refresh(cursor)}
            >
              Load more links
            </Button>
          </div>
        )}
      </section>
      <p className="library-note">
        Disabling a link stops redirects immediately. You can enable it again at
        any time. Expired links stay in your library.
      </p>
    </>
  );
}
