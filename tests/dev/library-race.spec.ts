import { test, expect } from "@playwright/test";

test("a late Strict Mode list response cannot undo a successful disable", async ({
  page,
}) => {
  const link = {
    code: "race-link",
    url: "https://example.com",
    enabled: true,
    createdAt: "2026-10-25T00:00:00Z",
  };
  let requests = 0;
  let release!: () => void;
  let finished!: () => void;
  const delayed = new Promise<void>((resolve) => {
    release = resolve;
  });
  const completed = new Promise<void>((resolve) => {
    finished = resolve;
  });
  const headers = { "access-control-allow-origin": "http://localhost:3001" };
  await page.route("http://127.0.0.1:8080/api/links**", async (route) => {
    if (route.request().method() === "PATCH") {
      return route.fulfill({ headers, json: { ...link, enabled: false } });
    }
    const first = ++requests === 1;
    if (first) await delayed;
    await route.fulfill({ headers, json: { links: [link] } });
    if (first) finished();
  });
  try {
    await page.goto("/dashboard/");
    const row = page.getByRole("article", {
      name: "Link race-link",
      exact: true,
    });
    await row
      .getByRole("button", { name: "Disable link", exact: true })
      .click();
    await expect(row.getByText("Disabled", { exact: true })).toBeVisible();
    expect(requests).toBe(2);
    release();
    await completed;
    await page.evaluate(
      () =>
        new Promise<void>((resolve) => {
          requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
        }),
    );
    await expect(row.getByText("Disabled", { exact: true })).toBeVisible();
  } finally {
    release();
  }
});
