#!/usr/bin/env bash
# Only run in an explicitly authorized AWS sandbox. Never edits an alias.
set -euo pipefail
[[ $# == 2 && ( $2 == success || $2 == rollback ) && ${E2E_CANARY_SANDBOX:-} == 1 ]] || {
  printf 'Usage: E2E_CANARY_SANDBOX=1 canary.sh outputs.json success|rollback (authorized sandbox only)\n' >&2; exit 1;
}
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
outputs=$1
scenario=$2
function_name=$(jq -er '.resolve_function_name.value' "$outputs")
target=$(jq -er '.resolve_published_version.value' "$outputs")
[[ $function_name =~ ^[a-z][a-z0-9-]+-resolve-link$ && $target =~ ^[1-9][0-9]*$ ]] || exit 1
alias_arn=$(jq -er '.lambda_aliases.value["resolve-link"]' "$outputs")
region=${AWS_REGION:-ap-southeast-1}
[[ ${E2E_CANARY_ACCOUNT_ID:-} =~ ^[0-9]{12}$ && $alias_arn == "arn:aws:lambda:$region:$E2E_CANARY_ACCOUNT_ID:function:$function_name:live" ]] || { printf 'Pin the authorized sandbox account and matching output alias ARN\n' >&2; exit 1; }
[[ $(aws sts get-caller-identity --query Account --output text) == "$E2E_CANARY_ACCOUNT_ID" ]] || exit 1
base=$(jq -er '.http_api_endpoint.value' "$outputs")
[[ $base =~ ^https://[a-z0-9]+\.execute-api\.[a-z0-9-]+\.amazonaws\.com$ && ${E2E_REDIRECT_CODE:-} =~ ^[A-Za-z0-9]{1,32}$ ]] || { printf 'Require a real test link and AWS HTTP API output\n' >&2; exit 1; }
work=$(mktemp -d)
runner=''
cleanup() {
  [[ -z $runner ]] || { kill "$runner" 2>/dev/null || true; wait "$runner" 2>/dev/null || true; }
  rm -rf "$work"
}
trap cleanup EXIT
aws lambda get-alias --function-name "$function_name" --name live --output json > "$work/before.json"
current=$(jq -er '.FunctionVersion' "$work/before.json")
[[ $current != "$target" ]] || { printf 'E2E requires a distinct published target version\n' >&2; exit 1; }
# Prove no-op using the actual alias, then test the distinct target.
jq --arg current "$current" '.resolve_published_version.value = $current' "$outputs" > "$work/same.json"
bash "$repo_root/scripts/deploy-canary.sh" "$work/same.json"
if [[ $scenario == success ]]; then
  bash "$repo_root/scripts/deploy-canary.sh" "$outputs"
else
  # Operator must provision an intentionally failing target via Terraform first.
  # No SNS alarm notifications or runtime settings are modified by this test.
  bash "$repo_root/scripts/deploy-canary.sh" "$outputs" > "$work/deployment.log" 2>&1 &
  runner=$!
  observed=false
  for ((attempt=0; attempt<120; attempt++)); do
    kill -0 "$runner" 2>/dev/null || break
    alias_json=$(aws lambda get-alias --function-name "$function_name" --name live --output json)
    if jq -e --arg target "$target" '(.RoutingConfig.AdditionalVersionWeights // {})[$target] > 0' <<<"$alias_json" >/dev/null; then
      observed=true
      # Function-wide Errors alarm observes this deliberately failing version.
      aws lambda invoke --function-name "$function_name" --qualifier "$target" --payload '{}' --cli-binary-format raw-in-base64-out "$work/result.json" > "$work/invoke.json"
      jq -e '.FunctionError | type == "string"' "$work/invoke.json" >/dev/null
      break
    fi
    sleep 5
  done
  status=0
  wait "$runner" || status=$?
  runner=''
  cat "$work/deployment.log"
  [[ $observed == true && $status != 0 ]] || { printf 'Fault/rollback was not observed\n' >&2; exit 1; }
  grep -q 'rollback .* completed' "$work/deployment.log"
fi
aws lambda get-alias --function-name "$function_name" --name live --output json > "$work/after.json"
expected=$target
[[ $scenario != rollback ]] || expected=$current
jq -e --arg expected "$expected" '.FunctionVersion == $expected and ((.RoutingConfig.AdditionalVersionWeights // {}) | length == 0)' "$work/after.json" >/dev/null
printf 'Canary %s verified: alias live=%s with no weighted routing\n' "$scenario" "$expected"

status=$(curl --silent --show-error --max-time 15 --max-redirs 0 -o /dev/null -D "$work/headers" -w '%{http_code}' "$base/$E2E_REDIRECT_CODE")
[[ $status == 302 ]] && grep -Eiq '^location: https?://' "$work/headers"
printf 'Real API redirect after %s PASS\n' "$scenario"
