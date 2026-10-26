import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/dev",
  retries: 0,
  use: { baseURL: "http://localhost:3001", browserName: "chromium" },
  webServer: {
    command:
      "pnpm --filter @shortly/web exec next dev --hostname localhost --port 3001",
    url: "http://localhost:3001",
    reuseExistingServer: false,
    timeout: 30000,
  },
});
