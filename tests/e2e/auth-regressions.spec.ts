import { test, expect } from "@playwright/test";
import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { parseEnv } from "node:util";

const api = "http://127.0.0.1:8080";
const link = {
  code: "private-link",
  url: "https://example.com/private",
  enabled: true,
  version: 1,
  createdAt: "2026-10-26T00:00:00Z",
};

for (const pending of ["GET", "PATCH"]) {
  test(`sign-out clears private library state despite a late ${pending} response`, async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "Sign in", exact: true }).click();
    await expect(page.getByRole("button", { name: "Sign out" })).toBeVisible();
    let delay = false;
    let release!: () => void;
    const delayed = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route(`${api}/api/links**`, async (route) => {
      if (!route.request().headers().authorization) {
        return route.fulfill({
          status: 401,
          json: { message: "Unauthorized" },
        });
      }
      if (delay && route.request().method() === pending) await delayed;
      await route.fulfill({
        json:
          route.request().method() === "PATCH"
            ? { ...link, enabled: false, version: 2 }
            : { links: [link], nextCursor: "private-cursor" },
      });
    });
    try {
      await page
        .getByRole("link", { name: "Link library", exact: true })
        .click();
      await expect(page.getByRole("article")).toHaveCount(1);
      delay = true;
      const requestStarted = page.waitForRequest(
        (request) =>
          request.url().startsWith(`${api}/api/links`) &&
          request.method() === pending,
      );
      await page
        .getByRole("button", {
          name: pending === "GET" ? "Refresh links" : "Disable link",
          exact: true,
        })
        .click();
      await requestStarted;
      await page.getByRole("button", { name: "Sign out" }).click();
      await expect(
        page.getByRole("button", { name: "Sign in", exact: true }).first(),
      ).toBeVisible();
      await expect(page.getByRole("article")).toHaveCount(0);
      const responseFinished = page.waitForResponse(
        (response) =>
          response.url().startsWith(`${api}/api/links`) &&
          response.request().method() === pending &&
          response.status() === 200,
      );
      release();
      await (await responseFinished).finished();
      await expect(page.getByRole("article")).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Load more links" }),
      ).toHaveCount(0);
    } finally {
      release();
    }
  });
}

for (const expiresAt of [
  "2099-12-31T12:30:45.123Z",
  "2020-01-01T12:30:45.123Z",
]) {
  test(`editing only the destination preserves expiration ${expiresAt}`, async ({
    page,
  }) => {
    let current = { ...link, expiresAt };
    await page.route(`${api}/api/links**`, (route) => {
      if (route.request().method() === "PATCH") {
        current = {
          ...current,
          ...route.request().postDataJSON(),
          version: current.version + 1,
        };
        return route.fulfill({ json: current });
      }
      return route.fulfill({ json: { links: [current] } });
    });
    await page.goto("/dashboard/");
    await page.getByRole("button", { name: "Edit link", exact: true }).click();
    await page
      .getByLabel("Edit destination URL", { exact: true })
      .fill("https://example.com/updated");
    await page.getByRole("button", { name: "Save changes" }).click();
    await expect(page.getByText("v2", { exact: true })).toBeVisible();
    expect(current.expiresAt).toBe(expiresAt);

    await page.getByRole("button", { name: "Edit link", exact: true }).click();
    await page.getByRole("button", { name: "Clear expiration" }).click();
    await page.getByRole("button", { name: "Save changes" }).click();
    await expect(page.getByText("v3", { exact: true })).toBeVisible();
    expect(current.expiresAt).toBeNull();
  });
}

test("the example environment completes local PKCE with a matching issuer", async ({
  request,
}) => {
  const env = parseEnv(readFileSync("apps/web/.env.example", "utf8"));
  const verifier = "local-env-regression-verifier-01234567890123456789";
  const redirectURI = "http://localhost:3000/auth/callback/";
  const auth = await request.get(
    `${env.NEXT_PUBLIC_COGNITO_DOMAIN}/oauth2/authorize`,
    {
      maxRedirects: 0,
      params: {
        response_type: "code",
        client_id: env.NEXT_PUBLIC_COGNITO_CLIENT_ID,
        redirect_uri: redirectURI,
        state: "env-test",
        code_challenge_method: "S256",
        code_challenge: createHash("sha256")
          .update(verifier)
          .digest("base64url"),
      },
    },
  );
  expect(auth.status()).toBe(302);
  const callback = new URL(auth.headers().location);
  expect(callback.searchParams.get("state")).toBe("env-test");
  const token = await request.post(
    `${env.NEXT_PUBLIC_COGNITO_DOMAIN}/oauth2/token`,
    {
      form: {
        grant_type: "authorization_code",
        client_id: env.NEXT_PUBLIC_COGNITO_CLIENT_ID,
        code: callback.searchParams.get("code")!,
        code_verifier: verifier,
        redirect_uri: redirectURI,
      },
    },
  );
  expect(token.status()).toBe(200);
  const tokens = await token.json();
  const claims = JSON.parse(
    Buffer.from(tokens.id_token.split(".")[1], "base64url").toString(),
  );
  expect(claims.iss).toBe(env.NEXT_PUBLIC_COGNITO_ISSUER);
});
