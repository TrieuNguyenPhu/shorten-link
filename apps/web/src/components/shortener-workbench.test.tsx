import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ShortenerWorkbench } from "./shortener-workbench";

const link = {
  code: "abc1234",
  short_url: "https://npt-shortenlink.dev/link/abc1234",
  target_url: "https://example.com/docs",
  status: "active",
  created_at: "2026-07-26T00:00:00Z",
  expires_at: null,
};

const successResponse = () =>
  new Response(JSON.stringify({ data: link }), {
    status: 201,
    headers: { "Content-Type": "application/json" },
  });

async function finishRequest() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(500);
  });
}

describe("ShortenerWorkbench", () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let writeTextMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock = vi.fn();
    writeTextMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: writeTextMock },
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  function submit({
    url = "https://example.com/docs",
    alias = "",
    expiration = "",
  } = {}) {
    fireEvent.change(screen.getByLabelText("URL đích"), {
      target: { value: url },
    });
    if (alias) {
      fireEvent.change(screen.getByLabelText(/Alias/), {
        target: { value: alias },
      });
    }
    if (expiration) {
      fireEvent.change(screen.getByLabelText(/Thời hạn/), {
        target: { value: expiration },
      });
    }
    fireEvent.click(screen.getByRole("button", { name: "Tạo short link" }));
  }

  it("focuses the first invalid field and does not call the API", () => {
    render(<ShortenerWorkbench />);

    submit({ url: "ftp://example.com/file" });

    expect(fetchMock).not.toHaveBeenCalled();
    expect(
      screen.getByText("URL phải bắt đầu bằng http:// hoặc https://."),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("URL đích")).toHaveFocus();
  });

  it.each([
    {
      name: "embedded credentials",
      url: "https://user:secret@example.com/docs",
      message: "URL không được chứa thông tin đăng nhập.",
    },
    {
      name: "a URL over 2,048 characters",
      url: `https://example.com/${"a".repeat(2030)}`,
      message: "URL không được vượt quá 2.048 ký tự.",
    },
  ])("rejects $name before calling the API", ({ url, message }) => {
    render(<ShortenerWorkbench />);

    submit({ url });

    expect(fetchMock).not.toHaveBeenCalled();
    expect(screen.getByText(message)).toBeInTheDocument();
    expect(screen.getByLabelText("URL đích")).toHaveFocus();
  });

  it("sends optional fields and focuses the successful result", async () => {
    fetchMock.mockResolvedValue(successResponse());
    render(<ShortenerWorkbench />);

    submit({ alias: "abc1234", expiration: "30" });
    expect(
      screen.getByRole("button", { name: "Đang tạo link…" }),
    ).toBeDisabled();
    await finishRequest();

    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock.mock.calls[0][1]?.body).toBe(
      JSON.stringify({
        url: "https://example.com/docs",
        custom_alias: "abc1234",
        expires_in_days: 30,
      }),
    );
    expect(
      screen.getByRole("link", {
        name: "https://npt-shortenlink.dev/link/abc1234",
      }),
    ).toBeVisible();
    expect(document.querySelector(".workbench__result-panel")).toHaveFocus();
  });

  it("announces successful and failed clipboard writes", async () => {
    fetchMock.mockResolvedValue(successResponse());
    writeTextMock.mockResolvedValueOnce(undefined);
    render(<ShortenerWorkbench />);
    submit();
    await finishRequest();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sao chép link" }));
    });
    expect(writeTextMock).toHaveBeenCalledWith(link.short_url);
    expect(screen.getByText("Link đã được lưu vào clipboard.")).toBeVisible();

    writeTextMock.mockRejectedValueOnce(new Error("denied"));
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Đã sao chép" }));
    });
    expect(
      screen.getByText(/Không thể truy cập clipboard/),
    ).toBeInTheDocument();
  });

  it("shows the mapped API error and focuses the result panel", async () => {
    fetchMock.mockResolvedValue(
      new Response(
        JSON.stringify({
          error: {
            code: "custom_alias_conflict",
            message: "custom_alias is already in use",
          },
        }),
        { status: 409 },
      ),
    );
    render(<ShortenerWorkbench />);

    submit({ alias: "abc1234" });
    await finishRequest();

    expect(
      screen.getByText("Alias đã được sử dụng. Hãy chọn alias khác."),
    ).toBeInTheDocument();
    expect(document.querySelector(".workbench__result-panel")).toHaveFocus();
  });
});
