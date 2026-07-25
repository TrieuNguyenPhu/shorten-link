import type { NextConfig } from "next";

const legacyApiProxyTarget = (
  process.env.LEGACY_API_PROXY_TARGET ?? "http://localhost:3000"
).replace(/\/+$/, "");

const isDevelopment = process.env.NODE_ENV === "development";

const developmentRewrites =
  isDevelopment
    ? {
        async rewrites() {
          return [
            {
              source: "/api/:path*",
              destination: `${legacyApiProxyTarget}/api/:path*`,
            },
            {
              source: "/link/:path*",
              destination: `${legacyApiProxyTarget}/link/:path*`,
            },
          ];
        },
      }
    : {};

const nextConfig: NextConfig = {
  trailingSlash: true,
  skipTrailingSlashRedirect: true,
  ...(!isDevelopment ? { output: "export" as const } : {}),
  ...developmentRewrites,
};

export default nextConfig;
