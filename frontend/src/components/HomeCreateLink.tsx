"use client";
import { useEffect, useState } from "react";
import { getAccessToken } from "@/lib/auth";
import { getPublicConfig } from "@/lib/config";
import { CreateLinkForm } from "@/features/links/CreateLinkForm";
export function HomeCreateLink() { const [token, setToken] = useState<string | null>(null); useEffect(() => setToken(getAccessToken(window.sessionStorage)), []); if (!token) return null; return <div className="mt-16 scroll-mt-8" id="create-link"><CreateLinkForm config={getPublicConfig()} token={token} /></div>; }
