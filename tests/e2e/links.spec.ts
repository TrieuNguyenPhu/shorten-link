import { test, expect } from "@playwright/test";

const api = "http://127.0.0.1:8080";

function makeLocalBearer(
  sub: string,
  scopes = ["shortly-api/links.read", "shortly-api/links.write"],
  expOffsetSeconds = 3600,
): string {
  const header = Buffer.from(
    JSON.stringify({ alg: "none", typ: "JWT" }),
  ).toString("base64url");
  const payload = Buffer.from(
    JSON.stringify({
      sub,
      email: `${sub}@example.com`,
      token_use: "access",
      scope: scopes.join(" "),
      exp: Math.floor(Date.now() / 1000) + expOffsetSeconds,
    }),
  ).toString("base64url");
  return `Bearer ${header}.${payload}.local-sig`;
}

test("library follows pagination and retains links after update failures", async ({
  page,
}) => {
  const first = {
    code: "page-one",
    url: "https://example.com/first",
    enabled: true,
    version: 1,
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

test("PKCE sign-in, create, edit, version conflict, disable, delete tombstone and sign-out", async ({
  page,
  request,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const alias = `e2e-${Date.now()}-${test.info().project.name}`;

  // 1. Unauthenticated API calls are rejected with 401.
  const unauthCreate = await request.post(`${api}/api/links`, {
    data: { url: "https://example.com/unauth" },
  });
  expect(unauthCreate.status()).toBe(401);

  // 2. Sign in through the browser PKCE callback flow.
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /A little link/ }),
  ).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.getByRole("button", { name: "Sign in" }).first().click();
  await expect(page.getByRole("button", { name: "Sign out" })).toBeVisible();

  // Verify session is stored in sessionStorage, never localStorage.
  const storageCheck = await page.evaluate(() => ({
    sessionKeys: Object.keys(window.sessionStorage).filter((k) =>
      k.startsWith("oidc."),
    ).length,
    localKeys: Object.keys(window.localStorage).filter((k) =>
      k.startsWith("oidc."),
    ).length,
  }));
  expect(storageCheck.sessionKeys).toBeGreaterThan(0);
  expect(storageCheck.localKeys).toBe(0);

  await page.screenshot({
    path: `test-results/home-${test.info().project.name}.png`,
    fullPage: true,
  });

  // 3. Create a short link as the authenticated user.
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

  const redirect1 = await request.get(`${api}/r/${alias}`, {
    maxRedirects: 0,
  });
  expect(redirect1.status()).toBe(302);
  expect(redirect1.headers().location).toBe(
    "https://example.com/docs?from=shortly#intro",
  );

  // Duplicate alias is rejected with 409.
  await page.getByRole("button", { name: "Create short link" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "already in use",
  );

  // 4. Open Link Library and edit destination URL.
  await page.getByRole("link", { name: "Link library", exact: true }).click();
  const row = page.getByRole("article", { name: `Link ${alias}` });
  await expect(row).toBeVisible();
  await expect(row.getByText("v1", { exact: true })).toBeVisible();

  await row.getByRole("button", { name: "Edit link" }).click();
  await row
    .getByLabel("Destination URL")
    .fill("https://example.com/updated-destination");
  await row.getByRole("button", { name: "Save changes" }).click();
  await expect(row.getByText("v2", { exact: true })).toBeVisible();
  await expect(
    row.getByText("https://example.com/updated-destination"),
  ).toBeVisible();

  const redirect2 = await request.get(`${api}/r/${alias}`, {
    maxRedirects: 0,
  });
  expect(redirect2.status()).toBe(302);
  expect(redirect2.headers().location).toBe(
    "https://example.com/updated-destination",
  );

  // 5. Simulate concurrent edit from another tab/session to trigger 409 version_conflict.
  await row.getByRole("button", { name: "Edit link" }).click();
  const concurrentUpdate = await request.patch(`${api}/api/links/${alias}`, {
    headers: { Authorization: makeLocalBearer("local-user-a") },
    data: {
      url: "https://example.com/concurrent-winner",
      version: 2,
    },
  });
  expect(concurrentUpdate.status()).toBe(200);

  await row
    .getByLabel("Destination URL")
    .fill("https://example.com/stale-attempt");
  await row.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "modified in another session",
  );
  await page.getByRole("button", { name: "Reload links" }).click();
  await expect(row.getByText("v3", { exact: true })).toBeVisible();
  await expect(
    row.getByText("https://example.com/concurrent-winner"),
  ).toBeVisible();

  // 6. Disable and re-enable link.
  await row.getByRole("button", { name: "Disable link" }).click();
  await expect(row.getByText("Disabled", { exact: true })).toBeVisible();
  expect(
    (await request.get(`${api}/r/${alias}`, { maxRedirects: 0 })).status(),
  ).toBe(410);
  await row.getByRole("button", { name: "Enable link" }).click();
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

  // 7. Delete link with confirmation -> tombstone prevents redirect (410) and alias reuse (409).
  await row.getByRole("button", { name: "Delete link" }).click();
  await row.getByRole("button", { name: "Confirm delete" }).click();
  await expect(row).toHaveCount(0);
  expect(
    (await request.get(`${api}/r/${alias}`, { maxRedirects: 0 })).status(),
  ).toBe(410);
  const reuseDeleted = await request.post(`${api}/api/links`, {
    headers: { Authorization: makeLocalBearer("local-user-a") },
    data: { url: "https://example.com/reuse", alias },
  });
  expect(reuseDeleted.status()).toBe(409);

  // 8. Sign out clears session.
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(
    page.getByRole("button", { name: "Sign in" }).first(),
  ).toBeVisible();
  expect(errors).toEqual([]);
});

test("multi-user A/B ownership isolation, scope enforcement, and expired links", async ({
  request,
}) => {
  const userA = makeLocalBearer(`user-a-${Date.now()}`);
  const userB = makeLocalBearer(`user-b-${Date.now()}`);
  const readOnlyToken = makeLocalBearer(`user-ro-${Date.now()}`, [
    "shortly-api/links.read",
  ]);
  const expiredToken = makeLocalBearer(
    `user-exp-${Date.now()}`,
    ["shortly-api/links.read", "shortly-api/links.write"],
    -60,
  );

  // Expired token -> 401; read-only token on write route -> 403.
  expect(
    (
      await request.get(`${api}/api/links`, {
        headers: { Authorization: expiredToken },
      })
    ).status(),
  ).toBe(401);
  expect(
    (
      await request.post(`${api}/api/links`, {
        headers: { Authorization: readOnlyToken },
        data: { url: "https://example.com/ro" },
      })
    ).status(),
  ).toBe(403);

  // User A creates a link.
  const createdA = await request.post(`${api}/api/links`, {
    headers: { Authorization: userA },
    data: {
      url: "https://example.com/user-a-only",
      expiresAt: new Date(Date.now() + 2500).toISOString(),
    },
  });
  expect(createdA.status()).toBe(201);
  const linkA = await createdA.json();
  expect(linkA.code).toMatch(/^[A-Za-z0-9_-]{8}$/);

  // User B cannot view, update, or delete User A's link (404 not_found).
  expect(
    (
      await request.get(`${api}/api/links/${linkA.code}`, {
        headers: { Authorization: userB },
      })
    ).status(),
  ).toBe(404);
  expect(
    (
      await request.patch(`${api}/api/links/${linkA.code}`, {
        headers: { Authorization: userB },
        data: { enabled: false, version: 1 },
      })
    ).status(),
  ).toBe(404);
  expect(
    (
      await request.delete(`${api}/api/links/${linkA.code}?version=1`, {
        headers: { Authorization: userB },
      })
    ).status(),
  ).toBe(404);

  // User B's list does not include User A's link.
  const listB = await (
    await request.get(`${api}/api/links`, {
      headers: { Authorization: userB },
    })
  ).json();
  expect(listB.links).toEqual([]);

  // Forged cursor from another owner is rejected with 400.
  const forgedCursor = Buffer.from(
    JSON.stringify({
      PK: `LINK#${linkA.code}`,
      GSI1PK: "OWNER#someone-else",
      GSI1SK: `2026-10-25T00:00:00Z#${linkA.code}`,
    }),
  ).toString("base64url");
  expect(
    (
      await request.get(`${api}/api/links?cursor=${forgedCursor}`, {
        headers: { Authorization: userB },
      })
    ).status(),
  ).toBe(400);

  // Public redirect works while active, then returns 410 after expiry.
  expect(
    (await request.get(`${api}/r/${linkA.code}`, { maxRedirects: 0 })).status(),
  ).toBe(302);
  await expect
    .poll(async () =>
      (
        await request.get(`${api}/r/${linkA.code}`, { maxRedirects: 0 })
      ).status(),
    )
    .toBe(410);

  const invalid = await request.post(`${api}/api/links`, {
    headers: { Authorization: userA },
    data: { url: "javascript:alert(1)" },
  });
  expect(invalid.status()).toBe(400);
  expect((await invalid.json()).error.code).toBe("invalid_input");
});

test("empty, loading, callback error and network error states are accessible", async ({
  page,
}) => {
  await page.goto(
    "/auth/callback/?error=access_denied&error_description=Sign-in+cancelled+by+user",
  );
  await expect(page.getByRole("main").getByRole("alert")).toContainText(
    "Sign-in cancelled by user",
  );
  await expect(
    page.getByRole("button", { name: "Try signing in again" }),
  ).toBeVisible();

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
  await page.route(`${api}/api/links`, (route) =>
    route.fulfill({ json: { links: [] } }),
  );
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
