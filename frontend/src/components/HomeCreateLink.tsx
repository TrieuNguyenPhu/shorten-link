"use client";
import { useEffect, useState } from "react";
import { getAccessToken } from "@/lib/auth";
import { getPublicConfig, type PublicConfig } from "@/lib/config";
import { CreateLinkForm } from "@/features/links/CreateLinkForm";
export function HomeCreateLink() {
  const [session, setSession] = useState<{ token: string; config: PublicConfig } | null>(null);
  useEffect(() => {
    try {
      const token = getAccessToken(window.sessionStorage);
      if (!token) return;
      setSession({ token, config: getPublicConfig() });
    } catch {
      setSession(null);
    }
  }, []);
  if (!session) return null;
  return <div className="mt-16 scroll-mt-8" id="create-link"><CreateLinkForm config={session.config} token={session.token} /></div>;
}
