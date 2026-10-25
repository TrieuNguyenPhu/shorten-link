import { test, expect } from "@playwright/test";

const api = "http://127.0.0.1:8080";

test("library follows pagination and retains links after update failures", async ({
  page,
}) => {
  const first = {
    code: "page-one",
    url: "https://example.com/first",
    enabled: true,
    createdAt: "2026-10-24T00:00:00Z",
  };
  const second = {
    ...first,
    code: "page-two",
    url: "https://example.com/second",
  };
  await page.route(`${api}/api/links**`, (route) => {
    if (route.request().method() === "PATCH")
      return route.fulfill({
        status: 500,
        json: {
          error: { code: "internal_error", message: "Unable to update link." },
        },
      });
    return route.fulfill({
      json: route.request().url().includes("cursor=")
        ? { links: [first, second] }
        : { links: [first], nextCursor: "next-page" },
    });
  });
  await page.goto("/dashboard/");
  await expect(page.getByRole("article")).toHaveCount(1);
  await page.getByRole("button", { name: "Load more links" }).click();
  await expect(page.getByRole("article")).toHaveCount(2);
  await expect(
    page.getByRole("button", { name: "Load more links" }),
  ).toHaveCount(0);
  const row = page.getByRole("article", { name: "Link page-one" });
  await row.getByRole("button", { name: "Disable link" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "Unable to update link.",
  );
  await expect(row.getByText("Active", { exact: true })).toBeVisible();
});

test("create, retrieve, redirect and manage a real persisted link", async ({
  page,
  request,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const alias = `e2e-${Date.now()}-${test.info().project.name}`;
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /A little link/ }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: `test-results/home-${test.info().project.name}.png`,
    fullPage: true,
  });
  await page
    .getByLabel("Destination URL")
    .fill("https://example.com/docs?from=shortly#intro");
  await page.getByLabel("Custom alias").fill(alias);
  await page.getByRole("button", { name: "Create short link" }).click();
  await expect(page.getByRole("status")).toContainText("Your link is ready");
  await expect(page.getByTestId("created-link")).toHaveAttribute(
    "href",
    `${api}/r/${alias}`,
  );
  const info = await request.get(`${api}/api/links/${alias}`);
  expect(info.status()).toBe(200);
  expect((await info.json()).enabled).toBe(true);
  const redirect = await request.get(`${api}/r/${alias}`, { maxRedirects: 0 });
  expect(redirect.status()).toBe(302);
  expect(redirect.headers().location).toBe(
    "https://example.com/docs?from=shortly#intro",
  );
  await page.getByRole("button", { name: "Create short link" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "already in use",
  );
  await page.getByRole("link", { name: "Link library", exact: true }).click();
  const row = page.getByRole("article", { name: `Link ${alias}` });
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: "Disable link" }).click();
  await expect(row.getByText("Disabled", { exact: true })).toBeVisible();
  expect(
    (await request.get(`${api}/r/${alias}`, { maxRedirects: 0 })).status(),
  ).toBe(410);
  await row.getByRole("button", { name: "Enable link" }).click();
  await expect(row.getByText("Active", { exact: true })).toBeVisible();
  await page.reload();
  await expect(row.getByText("Active", { exact: true })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: `test-results/library-${test.info().project.name}.png`,
    fullPage: true,
  });
  expect(errors).toEqual([]);
});

test("generated links expire and invalid destinations are rejected", async ({
  request,
}) => {
  const created = await request.post(`${api}/api/links`, {
    data: {
      url: "https://example.com/expiring",
      expiresAt: new Date(Date.now() + 2000).toISOString(),
    },
  });
  expect(created.status()).toBe(201);
  const link = await created.json();
  expect(link.code).toMatch(/^[A-Za-z0-9_-]{8}$/);
  expect(
    (await request.get(`${api}/r/${link.code}`, { maxRedirects: 0 })).status(),
  ).toBe(302);
  await expect
    .poll(async () =>
      (
        await request.get(`${api}/r/${link.code}`, { maxRedirects: 0 })
      ).status(),
    )
    .toBe(410);
  const invalid = await request.post(`${api}/api/links`, {
    data: { url: "javascript:alert(1)" },
  });
  expect(invalid.status()).toBe(400);
  expect((await invalid.json()).error.code).toBe("invalid_input");
});

test("empty, loading and network error states are accessible", async ({
  page,
}) => {
  await page.route(`${api}/api/links`, async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 350));
    await route.fulfill({ json: { links: [] } });
  });
  await page.goto("/dashboard/");
  await expect(
    page.getByRole("status").filter({ hasText: "Loading links" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Your next good link starts here" }),
  ).toBeVisible();
  await page.unroute(`${api}/api/links`);
  await page.route(`${api}/api/links`, (route) =>
    route.abort("connectionrefused"),
  );
  await page.getByRole("button", { name: "Refresh links" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "Cannot reach the local API",
  );
  await page.unroute(`${api}/api/links`);
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toHaveCount(0);
});

test("create form reports API failures and expiration validation", async ({
  page,
}) => {
  await page.goto("/");
  await page.getByLabel("Destination URL").fill("https://example.com");
  await page.getByLabel("Expiration (optional)").fill("2020-01-01T12:00");
  await page.getByRole("button", { name: "Create short link" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "future",
  );
  await page.getByLabel("Expiration (optional)").fill("");
  await page.route(`${api}/api/links`, (route) =>
    route.abort("connectionrefused"),
  );
  await page.getByRole("button", { name: "Create short link" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "Cannot reach the local API",
  );
});
