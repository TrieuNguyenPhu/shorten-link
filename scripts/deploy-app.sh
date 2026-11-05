#!/usr/bin/env bash
# Saved plans remain in the private state bucket, never in GitHub artifacts.
set -euo pipefail
umask 077
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
fail() { printf '%s\n' "$*" >&2; exit 1; }
[[ $# == 2 && ( $1 == plan || $1 == apply ) && $2 =~ ^[1-9][0-9]*$ ]] || fail 'Usage: deploy-app.sh plan|apply plan-run-id'
mode=$1
run_id=$2
: "${TF_STATE_BUCKET:?}" "${AWS_ACCOUNT_ID:?}" "${GITHUB_SHA:?}" "${GITHUB_REPOSITORY:?}"
[[ $AWS_ACCOUNT_ID =~ ^[0-9]{12}$ && $TF_STATE_BUCKET =~ ^[a-z0-9][a-z0-9-]+$ && $GITHUB_SHA =~ ^[0-9a-f]{40}$ ]] || fail 'Invalid deployment identity inputs'
[[ ${GITHUB_REF:-} == refs/heads/main && $GITHUB_REPOSITORY == TrieuNguyenPhu/shorten-link ]] || fail 'Only the original repository main branch may deploy'
tf_root=${DEPLOY_TF_ROOT:-"$repo_root/infrastructure/terraform/environments/dev"}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export AWS_PAGER=''
export TF_IN_AUTOMATION=true
[[ $(aws sts get-caller-identity --query Account --output text) == "$AWS_ACCOUNT_ID" ]] || fail 'Unexpected AWS account'
terraform -chdir="$tf_root" init -input=false -lockfile=readonly -backend-config="bucket=$TF_STATE_BUCKET" -backend-config="region=${AWS_REGION:-ap-southeast-1}"
terraform -chdir="$tf_root" output -json > "$work/before.json"
# Requires Phase 6.1 to be provisioned and prevents a second writer before apply.
bash scripts/deploy-canary.sh "$work/before.json" --check-only
prefix="s3://$TF_STATE_BUCKET/dev/plans/$run_id"
sha256sum backend/dist/*.zip > "$work/packages.sha256"
if [[ $mode == plan ]]; then
  terraform -chdir="$tf_root" plan -input=false -no-color -out="$work/app.tfplan"
  terraform -chdir="$tf_root" show -json "$work/app.tfplan" | jq '{resource_changes:[.resource_changes[]? | {address,actions:.change.actions}]}' > "$work/summary.json"
  # Values and provider configuration are omitted; the binary may contain secrets.
  cat "$work/summary.json"
  if [[ -n ${GITHUB_STEP_SUMMARY:-} ]]; then
    printf 'Saved plan `%s` for commit `%s`. Review resource actions before the apply dispatch.\n\n' "$run_id" "$GITHUB_SHA" >> "$GITHUB_STEP_SUMMARY"
    cat "$work/summary.json" >> "$GITHUB_STEP_SUMMARY"
  fi
  digest=$(sha256sum "$work/app.tfplan" | cut -d' ' -f1)
  jq -n --arg sha "$GITHUB_SHA" --arg repo "$GITHUB_REPOSITORY" --arg digest "$digest" --rawfile packages "$work/packages.sha256" \
    '{commit:$sha,repository:$repo,plan_sha256:$digest,packages:$packages}' > "$work/manifest.json"
  aws s3api put-object --bucket "$TF_STATE_BUCKET" --key "dev/plans/$run_id/app.tfplan" --body "$work/app.tfplan" --server-side-encryption AES256 --if-none-match '*' >/dev/null
  aws s3api put-object --bucket "$TF_STATE_BUCKET" --key "dev/plans/$run_id/manifest.json" --body "$work/manifest.json" --server-side-encryption AES256 --if-none-match '*' >/dev/null
  exit 0
fi
aws s3 cp "$prefix/manifest.json" "$work/manifest.json" --only-show-errors
jq -e --arg sha "$GITHUB_SHA" --arg repo "$GITHUB_REPOSITORY" --rawfile packages "$work/packages.sha256" \
  '.commit == $sha and .repository == $repo and .packages == $packages' "$work/manifest.json" >/dev/null || fail 'Saved plan commit/repository/packages mismatch; produce and review a fresh plan'
aws s3 cp "$prefix/app.tfplan" "$work/app.tfplan" --only-show-errors
[[ $(sha256sum "$work/app.tfplan" | cut -d' ' -f1) == $(jq -er .plan_sha256 "$work/manifest.json") ]] || fail 'Saved plan digest mismatch'
terraform -chdir="$tf_root" apply -input=false -no-color "$work/app.tfplan"
terraform -chdir="$tf_root" output -json > "$work/outputs.json"
# Build first; upload is gated on CodeDeploy success, including rollback settlement.
bash scripts/publish-frontend.sh "$tf_root" "$repo_root/frontend" --build-only
bash scripts/deploy-canary.sh "$work/outputs.json"
bash scripts/publish-frontend.sh "$tf_root" "$repo_root/frontend" --publish-only
node tests/e2e/frontend-hosting.mjs "$(jq -er '.frontend_url.value' "$work/outputs.json")"
aws s3 rm "$prefix/app.tfplan" --only-show-errors
aws s3 rm "$prefix/manifest.json" --only-show-errors
