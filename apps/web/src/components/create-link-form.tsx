"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { ArrowRight, Check, Link2, LoaderCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CopyButton } from "@/components/copy-button";
import { api, shortURL, type ShortLink } from "@/lib/api";

export function CreateLinkForm() {
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  const [created, setCreated] = useState<ShortLink | null>(null);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setCreated(null);
    const data = new FormData(event.currentTarget);
    const expires = String(data.get("expiresAt") || "");
    if (
      expires &&
      (!Number.isFinite(new Date(expires).getTime()) ||
        new Date(expires).getTime() <= Date.now())
    ) {
      setError("Expiration must be in the future.");
      return;
    }
    setPending(true);
    try {
      const link = await api.create({
        url: String(data.get("url")).trim(),
        alias: String(data.get("alias") || ""),
        ...(expires ? { expiresAt: new Date(expires).toISOString() } : {}),
      });
      setCreated(link);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Unable to create link. Please try again.",
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <section className="form-card" aria-labelledby="create-heading">
      <div className="card-heading">
        <span className="card-icon">
          <Link2 size={20} aria-hidden="true" />
        </span>
        <div>
          <h2 id="create-heading">Give your URL a little room.</h2>
          <p>Paste a destination. We’ll take it from here.</p>
        </div>
      </div>
      <form onSubmit={submit} aria-busy={pending}>
        <fieldset disabled={pending}>
          <label htmlFor="url">
            Destination URL <span className="required-dot">*</span>
          </label>
          <Input
            id="url"
            name="url"
            type="url"
            placeholder="https://example.com/something-great"
            required
            maxLength={2048}
            autoComplete="url"
            aria-describedby="url-help"
          />
          <p className="field-hint" id="url-help">
            The full http:// or https:// address you want to share.
          </p>
          <div className="form-grid">
            <div>
              <label htmlFor="alias">
                Custom alias <span>optional</span>
              </label>
              <Input
                id="alias"
                name="alias"
                placeholder="my-great-link"
                minLength={3}
                maxLength={32}
                pattern="[A-Za-z0-9_\-]{3,32}"
                aria-describedby="alias-help"
              />
              <p className="field-hint" id="alias-help">
                3–32 letters, numbers, - or _
              </p>
            </div>
            <div>
              <label htmlFor="expiresAt">
                Expiration <span>optional</span>
              </label>
              <Input
                id="expiresAt"
                name="expiresAt"
                type="datetime-local"
                aria-label="Expiration (optional)"
                aria-describedby="expiry-help"
              />
              <p className="field-hint" id="expiry-help">
                Your local time. Leave empty to keep it.
              </p>
            </div>
          </div>
          <Button type="submit" className="submit-button">
            {pending ? (
              <>
                <LoaderCircle className="animate-spin" aria-hidden="true" />
                Creating your link…
              </>
            ) : (
              <>
                Create short link <ArrowRight aria-hidden="true" />
              </>
            )}
          </Button>
        </fieldset>
        {error && (
          <p className="error-message" role="alert">
            {error}
          </p>
        )}
      </form>
      {created && (
        <div className="created-result" role="status">
          <p>
            <Check size={17} aria-hidden="true" />
            <strong>Your link is ready.</strong>
          </p>
          <div className="result-url">
            <a
              data-testid="created-link"
              href={shortURL(created.code)}
              target="_blank"
              rel="noopener noreferrer"
            >
              {shortURL(created.code)}
            </a>
            <CopyButton key={created.code} value={shortURL(created.code)} />
          </div>
          <Link href="/dashboard/">
            Find it in your link library{" "}
            <ArrowRight size={14} aria-hidden="true" />
          </Link>
        </div>
      )}
      <p className="form-footnote">
        One destination. One neat little link. All yours.
      </p>
    </section>
  );
}
