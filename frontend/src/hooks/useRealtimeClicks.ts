"use client";

import { useEffect, useRef, useState } from "react";

export type RealtimeStatus = "idle" | "connecting" | "open" | "reconnecting" | "closed";

export function mergeClickCount(current: number, incoming: number): number {
  return Math.max(current, incoming);
}

type Options = {
  code: string;
  websocketUrl: string;
  enabled?: boolean;
  onReconnect?: () => void;
};

type ClickMessage = { code?: unknown; clicks?: unknown };

export function useRealtimeClicks({ code, websocketUrl, enabled = true, onReconnect }: Options) {
  const [status, setStatus] = useState<RealtimeStatus>(enabled ? "connecting" : "idle");
  const [clicks, setClicks] = useState(0);
  const callbackRef = useRef(onReconnect);
  useEffect(() => {
    callbackRef.current = onReconnect;
  }, [onReconnect]);

  useEffect(() => {
    if (!enabled || !code || typeof WebSocket === "undefined") return;
    let socket: WebSocket | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let cancelled = false;
    let attempt = 0;

    const connect = () => {
      if (cancelled) return;
      setStatus(attempt === 0 ? "connecting" : "reconnecting");
      const base = websocketUrl.replace(/\/$/, "");
      socket = new WebSocket(`${base}?code=${encodeURIComponent(code)}`);
      socket.onopen = () => {
        attempt = 0;
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
        attempt += 1;
        const delay = Math.min(8000, 250 * 2 ** Math.min(attempt - 1, 5));
        setStatus("reconnecting");
        timer = setTimeout(connect, delay);
      };
    };

    connect();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
      socket?.close();
      setStatus("closed");
    };
  }, [code, enabled, websocketUrl]);

  return { clicks, status };
}
