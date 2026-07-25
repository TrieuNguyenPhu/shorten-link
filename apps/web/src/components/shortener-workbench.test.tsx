import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ShortenerWorkbench } from "./shortener-workbench";

const successfulResponse = () =>
  new Response(JSON.stringify({ short_url_code: "abc123" }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

async function finishMinimumLoadingDuration() {
  await act(async () => {
    await vi.advanceTimersByTimeAsync(416);
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
      value: {
        writeText: writeTextMock,
      },
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  function submitURL(value: string) {
    fireEvent.change(screen.getByLabelText("URL cần rút gọn"), {
      target: { value },
    });
    fireEvent.click(screen.getByRole("button", { name: "Rút gọn URL" }));
  }

  async function renderSuccessfulResult() {
    fetchMock.mockResolvedValue(successfulResponse());
    render(<ShortenerWorkbench />);
    submitURL("https://example.com/tai-lieu");
    await finishMinimumLoadingDuration();
  }

  it("blocks an invalid URL without creating a request", () => {
    render(<ShortenerWorkbench />);

    submitURL("ftp://example.com/file");

    expect(fetchMock).not.toHaveBeenCalled();
    expect(
      screen.getByText("URL phải bắt đầu bằng http:// hoặc https://."),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("URL cần rút gọn")).toHaveFocus();
  });

  it("renders the link from a legacy short_url_code response", async () => {
    fetchMock.mockResolvedValue(successfulResponse());
    render(<ShortenerWorkbench />);

    submitURL("https://example.com/tai-lieu");

    expect(
      screen.getByRole("button", { name: "Đang rút gọn…" }),
    ).toBeDisabled();
    await finishMinimumLoadingDuration();

    expect(fetchMock).toHaveBeenCalledOnce();
    expect(screen.getByRole("link", { name: "/link/abc123" })).toHaveAttribute(
      "href",
      "/link/abc123",
    );
    expect(document.querySelector(".result-panel")).toHaveFocus();
  });

  it.each([
    {
      name: "HTTP failure",
      response: () =>
        fetchMock.mockResolvedValue(
          new Response("Unavailable", { status: 503 }),
        ),
      message: "Python API trả về lỗi HTTP 503. Hãy thử lại.",
    },
    {
      name: "network failure",
      response: () => fetchMock.mockRejectedValue(new TypeError("offline")),
      message: "Không kết nối được Python API. Kiểm tra kết nối rồi thử lại.",
    },
  ])("renders a message for $name", async ({ response, message }) => {
    response();
    render(<ShortenerWorkbench />);

    submitURL("https://example.com/tai-lieu");
    await finishMinimumLoadingDuration();

    expect(screen.getByText(message)).toBeInTheDocument();
    expect(document.querySelector(".result-panel")).toHaveFocus();
  });

  it("announces a successful clipboard write", async () => {
    writeTextMock.mockResolvedValue(undefined);
    await renderSuccessfulResult();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sao chép link" }));
    });

    expect(writeTextMock).toHaveBeenCalledWith(
      "http://localhost:3001/link/abc123",
    );
    expect(screen.getByRole("button", { name: "Đã sao chép" })).toBeVisible();
    expect(
      screen.getByText("Short link đã được lưu vào clipboard."),
    ).toHaveAttribute("aria-live", "polite");
  });

  it("offers a manual fallback when clipboard access fails", async () => {
    writeTextMock.mockRejectedValue(new Error("denied"));
    await renderSuccessfulResult();

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Sao chép link" }));
    });

    expect(
      screen.getByRole("button", { name: "Thử sao chép lại" }),
    ).toBeVisible();
    expect(
      screen.getByText(
        "Không thể truy cập clipboard. Hãy chọn short link phía trên để sao chép thủ công.",
      ),
    ).toHaveAttribute("aria-live", "polite");
  });
});
