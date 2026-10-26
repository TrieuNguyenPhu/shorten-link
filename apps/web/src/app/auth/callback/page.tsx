"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AlertCircle, LoaderCircle, LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";
import { completeAuthCallback, useAuth } from "@/lib/auth";

export default function AuthCallbackPage() {
  const router = useRouter();
  const { signIn } = useAuth();
  const [error, setError] = useState("");
  const handled = useRef(false);

  useEffect(() => {
    if (handled.current) return;
    handled.current = true;

    completeAuthCallback(window.location.href)
      .then(({ returnTo }) => {
        router.replace(returnTo);
      })
      .catch((err: unknown) => {
        setError(
          err instanceof Error
            ? err.message
            : "Unable to verify your sign-in session. Please try signing in again.",
        );
      });
  }, [router]);

  return (
    <section className="form-card" aria-labelledby="callback-heading">
      <div className="card-heading">
        <span className="card-icon">
          {error ? (
            <AlertCircle size={20} aria-hidden="true" />
          ) : (
            <LoaderCircle
              className="animate-spin"
              size={20}
              aria-hidden="true"
            />
          )}
        </span>
        <div>
          <h1 id="callback-heading">
            {error ? "Could not complete sign-in" : "Completing sign-in…"}
          </h1>
          <p>
            {error
              ? "Your PKCE authorization exchange did not succeed."
              : "Verifying your Cognito PKCE session."}
          </p>
        </div>
      </div>
      {error ? (
        <div role="alert" className="callback-error-box">
          <p className="error-message">{error}</p>
          <div className="callback-actions">
            <Button
              type="button"
              onClick={() => void signIn({ returnTo: "/dashboard/" })}
            >
              <LogIn size={16} aria-hidden="true" />
              Try signing in again
            </Button>
            <Button variant="outline" asChild>
              <Link href="/">Return home</Link>
            </Button>
          </div>
        </div>
      ) : (
        <p className="loading-state" role="status">
          <LoaderCircle className="animate-spin" size={18} aria-hidden="true" />
          Establishing your session…
        </p>
      )}
    </section>
  );
}
