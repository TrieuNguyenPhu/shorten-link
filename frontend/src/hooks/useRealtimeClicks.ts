"use client";

import { useEffect, useRef, useState } from "react";

export type RealtimeStatus = "idle" | "connecting" | "open" | "reconnecting" | "closed";

export const MAX_RECONNECT_ATTEMPTS = 5;

export function calculateReconnectDelay(attempt: number, random = Math.random): number {
  const base = Math.min(8000, 250 * 2 ** Math.min(Math.max(attempt - 1, 0), 5));
  return Math.round(base * (0.5 + random() * 0.5));
}

export function mergeClickCount(current: number, incoming: number): number {
  return Math.max(current, incoming);
}

type Options = {
  code: string;
  realtimeTicket?: string;
  websocketUrl: string;
  enabled?: boolean;
  onReconnect?: () => void;
};

type ClickMessage = { code?: unknown; clicks?: unknown };

export function useRealtimeClicks({ code, realtimeTicket, websocketUrl, enabled = true, onReconnect }: Options) {
  const [status, setStatus] = useState<RealtimeStatus>(enabled && realtimeTicket ? "connecting" : "idle");
  const [clicks, setClicks] = useState(0);
  const callbackRef = useRef(onReconnect);
  useEffect(() => {
    callbackRef.current = onReconnect;
  }, [onReconnect]);

  useEffect(() => {
    if (!enabled || !code || !realtimeTicket || typeof WebSocket === "undefined") return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let cancelled = false;
    let attempt = 0;
    let openedAt = 0;
    const reconcileTimer = setInterval(() => callbackRef.current?.(), 30_000);

    const connect = () => {
      if (cancelled) return;
      setStatus(attempt === 0 ? "connecting" : "reconnecting");
      const base = websocketUrl.replace(/\/$/, "");
      socket = new WebSocket(`${base}?ticket=${encodeURIComponent(realtimeTicket)}`);
      socket.onopen = () => {
        attempt = 0;
        openedAt = Date.now();
        setStatus("open");
        callbackRef.current?.();
      };
      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(String(event.data)) as ClickMessage;
          if (message.code === code && typeof message.clicks === "number" && Number.isFinite(message.clicks)) {
            setClicks((current) => mergeClickCount(current, message.clicks as number));
          }
        } catch {
          // Ignore malformed best-effort pushes; the HTTP snapshot remains authoritative.
        }
      };
      socket.onerror = () => socket?.close();
      socket.onclose = () => {
        if (cancelled) return;
        if (openedAt && Date.now() - openedAt >= 10_000) attempt = 0;
        attempt += 1;
        if (attempt > MAX_RECONNECT_ATTEMPTS) {
          setStatus("closed");
          return;
        }
        const delay = calculateReconnectDelay(attempt);
        setStatus("reconnecting");
        timer = setTimeout(connect, delay);
      };
    };

    const recover = () => {
      if (!cancelled && socket?.readyState !== WebSocket.OPEN) {
        attempt = 0;
        if (timer) clearTimeout(timer);
        timer = setTimeout(connect, 0);
      } else {
        callbackRef.current?.();
      }
    };

    connect();
    window.addEventListener("online", recover);
    window.addEventListener("focus", recover);
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
      if (reconcileTimer) clearInterval(reconcileTimer);
      window.removeEventListener("online", recover);
      window.removeEventListener("focus", recover);
      socket?.close();
      setStatus("closed");
    };
  }, [code, enabled, realtimeTicket, websocketUrl]);

  return { clicks, status };
}
