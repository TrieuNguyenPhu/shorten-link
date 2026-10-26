"use client";

import Link from "next/link";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import {
  AlertTriangle,
  ArrowUpRight,
  Link2,
  LoaderCircle,
  Pencil,
  Plus,
  Power,
  RefreshCw,
  Trash2,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CopyButton } from "@/components/copy-button";
import { ApiError, api, shortURL, type ShortLink } from "@/lib/api";
import { useAuth } from "@/lib/auth";

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

function toDatetimeLocalValue(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (!Number.isFinite(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function LinkLibrary() {
  const { session } = useAuth();
  return <SessionLinkLibrary key={session?.sub ?? ""} />;
}

function SessionLinkLibrary() {
  const [links, setLinks] = useState<ShortLink[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState("");
  const [pendingCode, setPendingCode] = useState("");
  const [notice, setNotice] = useState("");
  const [now, setNow] = useState(0);
  const [editingCode, setEditingCode] = useState("");
  const [editURL, setEditURL] = useState("");
  const [editExpires, setEditExpires] = useState("");
  const [confirmDeleteCode, setConfirmDeleteCode] = useState("");
  const latestLoad = useRef<symbol | null>(null);

  const load = useCallback((nextCursor?: string) => {
    const request = Symbol();
    latestLoad.current = request;
    return api
      .list(nextCursor)
      .then((page) => {
        if (request !== latestLoad.current) return;
        setError("");
        setConflict("");
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
    setConflict("");
    setEditingCode("");
    setConfirmDeleteCode("");
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

  function handleMutationError(err: unknown, fallback: string) {
    if (err instanceof ApiError && err.code === "version_conflict") {
      setConflict(
        "This link was modified in another session. Reload your library to get the latest version before making changes.",
      );
      return;
    }
    setError(err instanceof Error ? err.message : fallback);
  }

  async function toggle(link: ShortLink) {
    setPendingCode(link.code);
    setError("");
    setConflict("");
    setNotice("");
    try {
      const updated = await api.setEnabled(
        link.code,
        !link.enabled,
        link.version ?? 1,
      );
      setLinks((previous) =>
        previous.map((item) => (item.code === updated.code ? updated : item)),
      );
      setNotice(`${updated.code} ${updated.enabled ? "enabled" : "disabled"}.`);
    } catch (err) {
      handleMutationError(err, "Unable to update link.");
    } finally {
      setPendingCode("");
    }
  }

  function startEdit(link: ShortLink) {
    setEditingCode(link.code);
    setConfirmDeleteCode("");
    setEditURL(link.url);
    setEditExpires(toDatetimeLocalValue(link.expiresAt));
    setError("");
    setConflict("");
  }

  async function submitEdit(
    event: FormEvent<HTMLFormElement>,
    link: ShortLink,
  ) {
    event.preventDefault();
    setError("");
    setConflict("");
    setNotice("");
    const expirationChanged =
      editExpires !== toDatetimeLocalValue(link.expiresAt);
    if (
      expirationChanged &&
      editExpires &&
      (!Number.isFinite(new Date(editExpires).getTime()) ||
        new Date(editExpires).getTime() <= Date.now())
    ) {
      setError("Expiration must be in the future.");
      return;
    }
    setPendingCode(link.code);
    try {
      const updated = await api.update(link.code, {
        version: link.version ?? 1,
        url: editURL.trim(),
        ...(expirationChanged
          ? {
              expiresAt: editExpires
                ? new Date(editExpires).toISOString()
                : null,
            }
          : {}),
      });
      setLinks((previous) =>
        previous.map((item) => (item.code === updated.code ? updated : item)),
      );
      setEditingCode("");
      setNotice(`${updated.code} updated.`);
    } catch (err) {
      handleMutationError(err, "Unable to save changes.");
    } finally {
      setPendingCode("");
    }
  }

  async function confirmDelete(link: ShortLink) {
    setPendingCode(link.code);
    setError("");
    setConflict("");
    setNotice("");
    try {
      await api.remove(link.code, link.version ?? 1);
      setLinks((previous) =>
        previous.filter((item) => item.code !== link.code),
      );
      setConfirmDeleteCode("");
      setEditingCode("");
      setNotice(`${link.code} deleted.`);
    } catch (err) {
      handleMutationError(err, "Unable to delete link.");
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
        {conflict && (
          <div className="error-banner conflict-banner" role="alert">
            <span>
              <AlertTriangle size={16} aria-hidden="true" /> {conflict}
            </span>
            <Button
              variant="outline"
              size="sm"
              disabled={loading}
              onClick={() => refresh()}
            >
              Reload links
            </Button>
          </div>
        )}
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
            const isEditing = editingCode === link.code;
            const isConfirmingDelete = confirmDeleteCode === link.code;
            const isBusy = pendingCode === link.code;
            return (
              <article
                className="link-row"
                key={link.code}
                aria-label={`Link ${link.code}`}
              >
                <div className="link-row-main">
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
                      <span className="version-badge">
                        v{link.version ?? 1}
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
                      onClick={() =>
                        isEditing ? setEditingCode("") : startEdit(link)
                      }
                      aria-label="Edit link"
                    >
                      <Pencil aria-hidden="true" />
                      Edit
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={!!pendingCode || loading}
                      onClick={() => void toggle(link)}
                      aria-label={link.enabled ? "Disable link" : "Enable link"}
                    >
                      {isBusy && !isEditing && !isConfirmingDelete ? (
                        <LoaderCircle
                          className="animate-spin"
                          aria-hidden="true"
                        />
                      ) : (
                        <Power aria-hidden="true" />
                      )}
                      {link.enabled ? "Disable" : "Enable"}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={!!pendingCode || loading}
                      onClick={() => {
                        setEditingCode("");
                        setConfirmDeleteCode(
                          isConfirmingDelete ? "" : link.code,
                        );
                      }}
                      aria-label="Delete link"
                    >
                      <Trash2 aria-hidden="true" />
                      Delete
                    </Button>
                  </div>
                </div>
                {isEditing && (
                  <form
                    className="edit-link-panel"
                    onSubmit={(e) => void submitEdit(e, link)}
                    aria-label={`Edit link ${link.code}`}
                  >
                    <div className="edit-link-grid">
                      <div>
                        <label htmlFor={`edit-url-${link.code}`}>
                          Destination URL
                        </label>
                        <Input
                          id={`edit-url-${link.code}`}
                          aria-label="Edit destination URL"
                          type="url"
                          required
                          maxLength={2048}
                          value={editURL}
                          onChange={(e) => setEditURL(e.target.value)}
                          disabled={isBusy}
                        />
                      </div>
                      <div>
                        <label htmlFor={`edit-expires-${link.code}`}>
                          Expiration <span>optional</span>
                        </label>
                        <div className="edit-expiry-row">
                          <Input
                            id={`edit-expires-${link.code}`}
                            aria-label="Edit expiration"
                            type="datetime-local"
                            value={editExpires}
                            onChange={(e) => setEditExpires(e.target.value)}
                            disabled={isBusy}
                          />
                          {editExpires && (
                            <Button
                              type="button"
                              variant="outline"
                              size="sm"
                              disabled={isBusy}
                              onClick={() => setEditExpires("")}
                            >
                              Clear expiration
                            </Button>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="edit-link-actions">
                      <Button type="submit" size="sm" disabled={isBusy}>
                        {isBusy ? (
                          <>
                            <LoaderCircle
                              className="animate-spin"
                              aria-hidden="true"
                            />
                            Saving…
                          </>
                        ) : (
                          "Save changes"
                        )}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={isBusy}
                        onClick={() => setEditingCode("")}
                      >
                        <X aria-hidden="true" />
                        Cancel
                      </Button>
                    </div>
                  </form>
                )}
                {isConfirmingDelete && (
                  <div
                    className="delete-confirm-panel"
                    role="region"
                    aria-label={`Confirm delete ${link.code}`}
                  >
                    <span>
                      Delete <strong>{link.code}</strong> permanently? This
                      short code cannot be reused.
                    </span>
                    <div className="delete-confirm-actions">
                      <Button
                        type="button"
                        variant="destructive"
                        size="sm"
                        disabled={isBusy}
                        onClick={() => void confirmDelete(link)}
                      >
                        {isBusy ? (
                          <>
                            <LoaderCircle
                              className="animate-spin"
                              aria-hidden="true"
                            />
                            Deleting…
                          </>
                        ) : (
                          "Confirm delete"
                        )}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={isBusy}
                        onClick={() => setConfirmDeleteCode("")}
                      >
                        Cancel
                      </Button>
                    </div>
                  </div>
                )}
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
        Disabling a link stops redirects immediately. Deleting a link removes
        its destination and permanently retires its short code.
      </p>
    </>
  );
}
