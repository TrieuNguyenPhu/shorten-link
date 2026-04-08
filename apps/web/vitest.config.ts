import { fileURLToPath } from "node:url";

import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    environment: "jsdom",
    environmentOptions: {
      jsdom: {
        pretendToBeVisual: true,
        url: "http://localhost:3001",
      },
    },
    setupFiles: "./vitest.setup.ts",
  },
});
