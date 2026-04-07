"use client";

import { FormEvent, useRef, useState } from "react";

import {
  createShortLink,
  ShortenerApiError,
  type ShortLinkResult,
} from "@/lib/shortener-api";

type RequestState = "idle" | "loading" | "success" | "error";
type CopyState = "idle" | "copying" | "copied" | "error";

const minimumLoadingDurationMilliseconds = 400;

function validateURL(value: string): string | null {
  const candidate = value.trim();
  if (!candidate) {
    return "Hãy nhập URL cần rút gọn.";
  }

  try {
    const parsed = new URL(candidate);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
      return "URL phải bắt đầu bằng http:// hoặc https://.";
    }
  } catch {
    return "URL chưa đúng định dạng. Ví dụ: https://example.com/tai-lieu";
  }

  return null;
}

export function ShortenerWorkbench() {
  const [url, setURL] = useState("");
  const [validationError, setValidationError] = useState<string | null>(null);
  const [requestError, setRequestError] = useState<string | null>(null);
  const [result, setResult] = useState<ShortLinkResult | null>(null);
  const [requestState, setRequestState] = useState<RequestState>("idle");
  const [copyState, setCopyState] = useState<CopyState>("idle");
  const requestInFlightRef = useRef(false);
  const urlInputRef = useRef<HTMLInputElement>(null);
  const resultPanelRef = useRef<HTMLDivElement>(null);

  function focusResultPanel() {
    window.requestAnimationFrame(() => resultPanelRef.current?.focus());
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (requestInFlightRef.current) {
      return;
    }

    const error = validateURL(url);
    setValidationError(error);
    if (error) {
      urlInputRef.current?.focus();
      return;
    }

    requestInFlightRef.current = true;
    setRequestState("loading");
    setRequestError(null);
    setResult(null);
    setCopyState("idle");
    const loadingStartedAt = performance.now();

    try {
      const created = await createShortLink(url.trim());
      const remainingLoadingTime = Math.max(
        0,
        minimumLoadingDurationMilliseconds -
          (performance.now() - loadingStartedAt),
      );
      if (remainingLoadingTime > 0) {
        await new Promise<void>((resolve) => {
          window.setTimeout(resolve, remainingLoadingTime);
        });
      }
      setResult(created);
      setRequestState("success");
      focusResultPanel();
    } catch (error) {
      const remainingLoadingTime = Math.max(
        0,
        minimumLoadingDurationMilliseconds -
          (performance.now() - loadingStartedAt),
      );
      if (remainingLoadingTime > 0) {
        await new Promise<void>((resolve) => {
          window.setTimeout(resolve, remainingLoadingTime);
        });
      }

      if (error instanceof ShortenerApiError) {
        setRequestError(
          error.kind === "http"
            ? `Python API trả về lỗi HTTP ${error.status}. Hãy thử lại.`
            : "Python API trả về dữ liệu không đúng định dạng JSON legacy.",
        );
      } else {
        setRequestError(
          "Không kết nối được Python API. Kiểm tra kết nối rồi thử lại.",
        );
      }
      setRequestState("error");
      focusResultPanel();
    } finally {
      requestInFlightRef.current = false;
    }
  }

  async function handleCopy() {
    if (!result || copyState === "copying") {
      return;
    }

    setCopyState("copying");
    try {
      const absoluteShortURL = new URL(result.path, window.location.origin);
      await navigator.clipboard.writeText(absoluteShortURL.toString());
      setCopyState("copied");
    } catch {
      setCopyState("error");
    }
  }

  return (
    <section className="task-surface" aria-labelledby="surface-title">
      <div className="task-surface__copy">
        <h2 id="surface-title">Tạo short link từ một URL.</h2>
        <p>
          Preview gửi đúng payload URL-only đến Python API hiện tại qua đường
          dẫn cùng origin.
        </p>

        <form
          id="shorten-form"
          className="shortener-form"
          onSubmit={handleSubmit}
          noValidate
        >
          <label htmlFor="target-url">URL cần rút gọn</label>
          <div className="shortener-form__controls">
            <input
              ref={urlInputRef}
              id="target-url"
              name="url"
              type="url"
              inputMode="url"
              autoComplete="url"
              placeholder="https://example.com/tai-lieu"
              value={url}
              onChange={(event) => {
                setURL(event.target.value);
                if (validationError) {
                  setValidationError(null);
                }
              }}
              aria-invalid={Boolean(validationError)}
              aria-describedby="target-url-help"
              required
            />
            <button
              type="submit"
              disabled={requestState === "loading"}
              aria-busy={requestState === "loading"}
              data-state={requestState}
            >
              {requestState === "loading" ? (
                <>
                  <span className="loading-spinner" aria-hidden="true" />
                  Đang rút gọn…
                </>
              ) : (
                "Rút gọn URL"
              )}
            </button>
          </div>
          <p
            id="target-url-help"
            className="field-help"
            data-error={Boolean(validationError)}
          >
            {validationError ??
              "Chấp nhận địa chỉ đầy đủ bắt đầu bằng http:// hoặc https://."}
          </p>
        </form>
      </div>

      <div
        ref={resultPanelRef}
        className="result-panel"
        data-state={requestState}
        aria-live="polite"
        aria-busy={requestState === "loading"}
        tabIndex={-1}
      >
        {requestState === "loading" ? (
          <div className="result-panel__content">
            <p className="result-panel__label">Đang gửi request</p>
            <h3>Python API đang tạo short link.</h3>
            <p>Giữ nguyên cửa sổ này trong khi request được xử lý.</p>
          </div>
        ) : requestState === "success" && result ? (
          <div className="result-panel__content">
            <p className="result-panel__label">Short link đã tạo</p>
            <a className="result-panel__link" href={result.path}>
              {result.path}
            </a>
            <button
              className="copy-button"
              type="button"
              onClick={handleCopy}
              disabled={copyState === "copying"}
              aria-busy={copyState === "copying"}
              data-state={copyState}
            >
              {copyState === "copying"
                ? "Đang sao chép…"
                : copyState === "copied"
                  ? "Đã sao chép"
                  : copyState === "error"
                    ? "Thử sao chép lại"
                    : "Sao chép link"}
            </button>
            <p
              className="copy-status"
              role="status"
              aria-live="polite"
              data-state={copyState}
            >
              {copyState === "copied"
                ? "Short link đã được lưu vào clipboard."
                : copyState === "error"
                  ? "Không thể truy cập clipboard. Hãy chọn short link phía trên để sao chép thủ công."
                  : null}
            </p>
            <p>
              Mã <code>{result.code}</code> được trả trực tiếp từ Python API.
            </p>
          </div>
        ) : requestState === "error" && requestError ? (
          <div className="result-panel__content" role="alert">
            <p className="result-panel__label">Không thể tạo link</p>
            <h3>Request chưa hoàn tất.</h3>
            <p>{requestError}</p>
          </div>
        ) : (
          <div className="result-panel__content">
            <p className="result-panel__label">
              POST /api/generate-short-url
            </p>
            <h3>Kết quả xuất hiện tại đây.</h3>
            <p>
              Preview đọc <code>short_url_code</code> và dựng đường dẫn{" "}
              <code>/link/&#123;code&#125;</code>.
            </p>
          </div>
        )}
      </div>
    </section>
  );
}
