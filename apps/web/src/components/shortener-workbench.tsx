"use client";

import { FormEvent, useState } from "react";

import {
  createShortLink,
  type ShortLinkResult,
} from "@/lib/shortener-api";

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

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const error = validateURL(url);
    setValidationError(error);
    if (error) {
      return;
    }

    setRequestError(null);
    setResult(null);

    try {
      setResult(await createShortLink(url.trim()));
    } catch {
      setRequestError("API hiện tại chưa thể tạo short link. Hãy thử lại.");
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
            <button type="submit">Rút gọn URL</button>
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

      <div className="result-panel" aria-live="polite">
        {result ? (
          <div className="result-panel__content">
            <p className="result-panel__label">Short link đã tạo</p>
            <a href={result.path}>{result.path}</a>
            <p>
              Mã <code>{result.code}</code> được trả trực tiếp từ Python API.
            </p>
          </div>
        ) : requestError ? (
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
