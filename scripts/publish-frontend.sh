#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tf_root=${1:-"$repo_root/infrastructure/terraform/environments/dev"}
frontend_root=${2:-"$repo_root/frontend"}
out="$frontend_root/out"
mode=${3:---all}
case "$mode" in --all|--build-only|--publish-only) ;; *) printf 'Unknown publish mode\n' >&2; exit 1 ;; esac
output() { terraform -chdir="$tf_root" output -raw "$1"; }

bucket=$(output frontend_bucket_name)
distribution=$(output cloudfront_distribution_id)
site=$(output frontend_url)
NEXT_PUBLIC_API_BASE_URL=$(output http_api_endpoint)
NEXT_PUBLIC_WS_URL=$(output websocket_endpoint)
NEXT_PUBLIC_COGNITO_DOMAIN=$(output cognito_login_domain)
NEXT_PUBLIC_COGNITO_CLIENT_ID=$(output cognito_client_id)
export NEXT_PUBLIC_API_BASE_URL NEXT_PUBLIC_WS_URL NEXT_PUBLIC_COGNITO_DOMAIN NEXT_PUBLIC_COGNITO_CLIENT_ID
export NEXT_PUBLIC_AUTH_REDIRECT_URI="$site/auth/callback/" NEXT_PUBLIC_AUTH_LOGOUT_URI="$site/"

if [[ $NEXT_PUBLIC_WS_URL != wss://*/prod || $site != https://*.cloudfront.net ]]; then
  printf 'Expected the deployed CloudFront origin and prod WebSocket endpoint.\n' >&2
  exit 1
fi
# Reuse the same validator as the browser; Node 26 executes erasable TypeScript.
node --input-type=module - "$repo_root/frontend/src/lib/config.ts" <<'JS'
import { pathToFileURL } from 'node:url';
const { getPublicConfig } = await import(pathToFileURL(process.argv[2]).href);
getPublicConfig();
JS
config_digest=$(printf '%s\n' "$NEXT_PUBLIC_API_BASE_URL" "$NEXT_PUBLIC_WS_URL" "$NEXT_PUBLIC_COGNITO_DOMAIN" "$NEXT_PUBLIC_COGNITO_CLIENT_ID" "$NEXT_PUBLIC_AUTH_REDIRECT_URI" "$NEXT_PUBLIC_AUTH_LOGOUT_URI" | sha256sum | cut -d' ' -f1)
if [[ $mode != --publish-only ]]; then
  corepack pnpm@10.33.2 --dir "$frontend_root" install --frozen-lockfile
  corepack pnpm@10.33.2 --dir "$frontend_root" build
  corepack pnpm@10.33.2 --dir "$frontend_root" test
  printf '%s\n' "$config_digest" > "$out/.publish-config"
fi
if [[ ! -f "$out/.publish-config" || $(cat "$out/.publish-config") != "$config_digest" ]]; then
  printf 'Artifact was not built with the current public deployment config.\n' >&2
  exit 1
fi
for route in index.html login/index.html auth/callback/index.html dashboard/index.html privacy/index.html; do
  if [[ ! -s "$out/$route" ]]; then
    printf 'Missing exported page %s\n' "$out/$route" >&2
    exit 1
  fi
done

[[ $mode != --build-only ]] || exit 0

# Keep previous hashed bundles available to clients still using the old HTML.
aws s3 sync "$out/_next/static/" "s3://$bucket/_next/static/" \
  --cache-control 'public,max-age=31536000,immutable' --only-show-errors
# Upload route data before publishing the HTML that references it. Always replace
# mutable keys so even a same-size change built from older source is published.
aws s3 cp "$out/" "s3://$bucket/" --recursive --exclude '_next/static/*' --exclude '*.html' --exclude '.publish-config' \
  --cache-control 'public,max-age=0,must-revalidate' --only-show-errors
aws s3 cp "$out/" "s3://$bucket/" --recursive --exclude '*' --include '*.html' \
  --cache-control 'public,max-age=0,must-revalidate' --content-type 'text/html; charset=utf-8' --only-show-errors

invalidation=$(aws cloudfront create-invalidation --distribution-id "$distribution" --paths '/*' \
  --query 'Invalidation.Id' --output text)
aws cloudfront wait invalidation-completed --distribution-id "$distribution" --id "$invalidation"
printf 'Published %s\n' "$site"
