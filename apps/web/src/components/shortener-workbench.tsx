"use client";

import { FormEvent, useEffect, useRef, useState } from "react";

import {
  createShortLink,
  ShortenerApiError,
  type ShortLink,
} from "@/lib/shortener-api";

type RequestState = "idle" | "loading" | "success" | "error";
type CopyState = "idle" | "copying" | "copied" | "error";

const aliasPattern = /^[a-z0-9-]{4,32}$/;
const spinnerDelayMilliseconds = 150;
const spinnerMinimumVisibleMilliseconds = 300;

const errorMessages: Record<string, string> = {
  invalid_url:
    "URL chưa hợp lệ. Hãy dùng một địa chỉ đầy đủ bắt đầu bằng http:// hoặc https://.",
  invalid_custom_alias:
    "Alias phải có 4–32 ký tự thường, chữ số hoặc dấu gạch ngang.",
  reserved_custom_alias:
    "Alias này dành cho hệ thống. Hãy chọn một tên khác.",
  invalid_expiration: "Thời hạn phải nằm trong khoảng 1–365 ngày.",
  custom_alias_conflict: "Alias đã được sử dụng. Hãy chọn alias khác.",
  code_generation_exhausted:
    "Hệ thống chưa cấp được mã ngắn. Hãy thử lại sau ít phút.",
  payload_too_large: "Dữ liệu gửi lên vượt giới hạn cho phép.",
  unsupported_media_type: "API chỉ nhận request JSON.",
};

function validateURL(value: string): string | null {
  if (!value.trim()) {
    return "Hãy nhập URL cần rút gọn.";
  }

  try {
    const parsed = new URL(value);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return "URL phải bắt đầu bằng http:// hoặc https://.";
    }
  } catch {
    return "URL chưa đúng định dạng. Ví dụ: https://example.com/tai-lieu";
  }

  return null;
}

function validateAlias(value: string): string | null {
  if (!value) {
    return null;
  }
  return aliasPattern.test(value)
    ? null
    : "Dùng 4–32 ký tự thường, chữ số hoặc dấu gạch ngang.";
}

function validateExpiration(value: string): string | null {
  if (!value) {
    return null;
  }
  const days = Number(value);
  return Number.isInteger(days) && days >= 1 && days <= 365
    ? null
    : "Chọn một số nguyên từ 1 đến 365.";
}

function formatDate(value: string | null): string {
  if (!value) {
    return "Không giới hạn";
  }
  return new Intl.DateTimeFormat("vi-VN", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

export function ShortenerWorkbench() {
  const [url, setURL] = useState("");
  const [alias, setAlias] = useState("");
  const [expiration, setExpiration] = useState("");
  const [touched, setTouched] = useState({
    url: false,
    alias: false,
    expiration: false,
  });
  const [requestState, setRequestState] = useState<RequestState>("idle");
  const [showSpinner, setShowSpinner] = useState(false);
  const [requestError, setRequestError] = useState<string | null>(null);
  const [result, setResult] = useState<ShortLink | null>(null);
  const [copyState, setCopyState] = useState<CopyState>("idle");
  const urlInputRef = useRef<HTMLInputElement>(null);
  const aliasInputRef = useRef<HTMLInputElement>(null);
  const expirationInputRef = useRef<HTMLInputElement>(null);
  const resultPanelRef = useRef<HTMLDivElement>(null);
  const requestRef = useRef<AbortController | null>(null);
  const copyTimerRef = useRef<number | null>(null);
  const spinnerDelayRef = useRef<number | null>(null);
  const spinnerMinimumRef = useRef<number | null>(null);
  const spinnerShownAtRef = useRef<number | null>(null);

  const urlError = touched.url ? validateURL(url) : null;
  const aliasError = touched.alias ? validateAlias(alias) : null;
  const expirationError = touched.expiration
    ? validateExpiration(expiration)
    : null;

  useEffect(() => {
    const focusURLField = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        urlInputRef.current?.focus();
      }
    };

    window.addEventListener("keydown", focusURLField);
    return () => {
      window.removeEventListener("keydown", focusURLField);
      requestRef.current?.abort();
      if (copyTimerRef.current !== null) {
        window.clearTimeout(copyTimerRef.current);
      }
      if (spinnerDelayRef.current !== null) {
        window.clearTimeout(spinnerDelayRef.current);
      }
      if (spinnerMinimumRef.current !== null) {
        window.clearTimeout(spinnerMinimumRef.current);
      }
    };
  }, []);

  function beginLoadingIndicator() {
    if (spinnerDelayRef.current !== null) {
      window.clearTimeout(spinnerDelayRef.current);
    }
    if (spinnerMinimumRef.current !== null) {
      window.clearTimeout(spinnerMinimumRef.current);
    }
    spinnerShownAtRef.current = null;
    setShowSpinner(false);
    spinnerDelayRef.current = window.setTimeout(() => {
      spinnerDelayRef.current = null;
      spinnerShownAtRef.current = performance.now();
      setShowSpinner(true);
    }, spinnerDelayMilliseconds);
  }

  async function finishLoadingIndicator() {
    if (spinnerDelayRef.current !== null) {
      window.clearTimeout(spinnerDelayRef.current);
      spinnerDelayRef.current = null;
      return;
    }

    if (spinnerShownAtRef.current === null) {
      return;
    }

    const remaining = Math.max(
      0,
      spinnerMinimumVisibleMilliseconds -
        (performance.now() - spinnerShownAtRef.current),
    );
    if (remaining > 0) {
      await new Promise<void>((resolve) => {
        spinnerMinimumRef.current = window.setTimeout(() => {
          spinnerMinimumRef.current = null;
          resolve();
        }, remaining);
      });
    }
    spinnerShownAtRef.current = null;
    setShowSpinner(false);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTouched({ url: true, alias: true, expiration: true });

    const nextURLError = validateURL(url);
    const nextAliasError = validateAlias(alias);
    const nextExpirationError = validateExpiration(expiration);
    if (nextURLError || nextAliasError || nextExpirationError) {
      setRequestState("error");
      setRequestError("Kiểm tra lại các trường được đánh dấu rồi thử lại.");
      if (nextURLError) {
        urlInputRef.current?.focus();
      } else if (nextAliasError) {
        aliasInputRef.current?.focus();
      } else {
        expirationInputRef.current?.focus();
      }
      return;
    }

    requestRef.current?.abort();
    const controller = new AbortController();
    requestRef.current = controller;
    setRequestState("loading");
    setRequestError(null);
    setCopyState("idle");
    beginLoadingIndicator();

    try {
      const created = await createShortLink(
        {
          url: url.trim(),
          ...(alias ? { custom_alias: alias } : {}),
          ...(expiration
            ? { expires_in_days: Number(expiration) }
            : {}),
        },
        controller.signal,
      );
      await finishLoadingIndicator();
      if (controller.signal.aborted || requestRef.current !== controller) {
        return;
      }
      setResult(created);
      setRequestState("success");
      window.requestAnimationFrame(() => resultPanelRef.current?.focus());
    } catch (error) {
      if (controller.signal.aborted) {
        return;
      }
      await finishLoadingIndicator();
      if (controller.signal.aborted || requestRef.current !== controller) {
        return;
      }
      const message =
        error instanceof ShortenerApiError
          ? (errorMessages[error.code] ??
            "API không thể tạo link lúc này. Kiểm tra backend và thử lại.")
          : "Không kết nối được API. Kiểm tra backend tại cổng 8080 rồi thử lại.";
      setRequestError(message);
      setRequestState("error");
      window.requestAnimationFrame(() => resultPanelRef.current?.focus());
    }
  }

  async function handleCopy() {
    if (!result) {
      return;
    }

    try {
      setCopyState("copying");
      await navigator.clipboard.writeText(result.short_url);
      setCopyState("copied");
      if (copyTimerRef.current !== null) {
        window.clearTimeout(copyTimerRef.current);
      }
      copyTimerRef.current = window.setTimeout(
        () => setCopyState("idle"),
        2500,
      );
    } catch {
      setCopyState("error");
    }
  }

  return (
    <section className="workbench" aria-labelledby="workbench-title">
      <div className="workbench__form-panel">
        <div className="workbench__heading">
          <div>
            <h2 id="workbench-title">Tạo một short link</h2>
            <p>Alias và thời hạn đều không bắt buộc.</p>
          </div>
          <span
            className="keyboard-hint"
            aria-label="Phím tắt Control hoặc Command K"
          >
            Ctrl / ⌘ K
          </span>
        </div>

        <form
          id="shorten-form"
          className="shortener-form"
          onSubmit={handleSubmit}
          noValidate
        >
          <div className="field field--wide">
            <label htmlFor="target-url">URL đích</label>
            <input
              ref={urlInputRef}
              id="target-url"
              name="url"
              type="url"
              inputMode="url"
              autoComplete="url"
              placeholder="https://example.com/tai-lieu"
              value={url}
              disabled={requestState === "loading"}
              onChange={(event) => setURL(event.target.value)}
              onBlur={() =>
                setTouched((current) => ({ ...current, url: true }))
              }
              aria-invalid={Boolean(urlError)}
              aria-describedby="target-url-help"
              required
            />
            <p
              id="target-url-help"
              className="field__help"
              data-error={Boolean(urlError)}
            >
              {urlError ??
                "Chấp nhận URL HTTP hoặc HTTPS, tối đa 2.048 ký tự."}
            </p>
          </div>

          <div className="field-row">
            <div className="field">
              <label htmlFor="custom-alias">
                Alias <span>tùy chọn</span>
              </label>
              <input
                ref={aliasInputRef}
                id="custom-alias"
                name="custom_alias"
                type="text"
                autoComplete="off"
                placeholder="tai-lieu-go"
                minLength={4}
                maxLength={32}
                pattern="[a-z0-9-]{4,32}"
                value={alias}
                disabled={requestState === "loading"}
                onChange={(event) =>
                  setAlias(event.target.value.toLowerCase())
                }
                onBlur={() =>
                  setTouched((current) => ({ ...current, alias: true }))
                }
                aria-invalid={Boolean(aliasError)}
                aria-describedby="custom-alias-help"
              />
              <p
                id="custom-alias-help"
                className="field__help"
                data-error={Boolean(aliasError)}
              >
                {aliasError ??
                  "4–32 ký tự thường, chữ số hoặc dấu gạch ngang."}
              </p>
            </div>

            <div className="field">
              <label htmlFor="expires-in-days">
                Thời hạn <span>tùy chọn</span>
              </label>
              <input
                ref={expirationInputRef}
                id="expires-in-days"
                name="expires_in_days"
                type="number"
                inputMode="numeric"
                placeholder="30"
                min={1}
                max={365}
                step={1}
                value={expiration}
                disabled={requestState === "loading"}
                onChange={(event) => setExpiration(event.target.value)}
                onBlur={() =>
                  setTouched((current) => ({
                    ...current,
                    expiration: true,
                  }))
                }
                aria-invalid={Boolean(expirationError)}
                aria-describedby="expires-in-days-help"
              />
              <p
                id="expires-in-days-help"
                className="field__help"
                data-error={Boolean(expirationError)}
              >
                {expirationError ?? "Để trống nếu link không cần hết hạn."}
              </p>
            </div>
          </div>

          <button
            className="primary-button"
            type="submit"
            disabled={requestState === "loading"}
            data-state={requestState}
            aria-busy={requestState === "loading"}
          >
            {requestState === "loading" ? (
              <>
                {showSpinner ? (
                  <span className="spinner" aria-hidden="true" />
                ) : null}
                Đang tạo link…
              </>
            ) : (
              "Tạo short link"
            )}
          </button>

          <p className="form-status" role="status" aria-live="polite">
            {requestError}
          </p>
        </form>
      </div>

      <div
        ref={resultPanelRef}
        className="workbench__result-panel"
        data-state={requestState}
        aria-live="polite"
        tabIndex={-1}
      >
        {result ? (
          <div className="result" key={result.code}>
            <p className="result__status">
              {requestState === "success"
                ? "Link đã sẵn sàng"
                : "Kết quả gần nhất"}
            </p>
            <a
              className="result__url"
              href={result.short_url}
              target="_blank"
              rel="noreferrer"
            >
              {result.short_url}
            </a>
            <button
              className="copy-button"
              type="button"
              onClick={handleCopy}
              data-state={copyState}
              disabled={copyState === "copying"}
              aria-busy={copyState === "copying"}
            >
              {copyState === "copying"
                ? "Đang sao chép…"
                : copyState === "copied"
                ? "Đã sao chép"
                : copyState === "error"
                  ? "Sao chép lại"
                  : "Sao chép link"}
            </button>
            <p
              className="copy-status"
              role="status"
              aria-live="polite"
              data-state={copyState}
            >
              {copyState === "copied"
                ? "Link đã được lưu vào clipboard."
                : copyState === "error"
                  ? "Không thể truy cập clipboard. Hãy chọn link phía trên để sao chép thủ công."
                  : null}
            </p>

            <dl className="result__meta">
              <div>
                <dt>Mã</dt>
                <dd>{result.code}</dd>
              </div>
              <div>
                <dt>URL đích</dt>
                <dd>{result.target_url}</dd>
              </div>
              <div>
                <dt>Hết hạn</dt>
                <dd>{formatDate(result.expires_at)}</dd>
              </div>
            </dl>
          </div>
        ) : (
          <div className="result-empty">
            <p className="result-empty__index">POST /api/v1/links</p>
            <h3>Kết quả xuất hiện tại đây.</h3>
            <p>
              Short URL, mã, URL đích và thời hạn được trả trực tiếp từ API —
              không dựng dữ liệu mẫu.
            </p>
          </div>
        )}
      </div>
    </section>
  );
}
