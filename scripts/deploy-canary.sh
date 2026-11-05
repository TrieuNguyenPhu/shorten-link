#!/usr/bin/env bash
# The deployment workflow is the sole writer; never call update-alias here.
set -euo pipefail
export AWS_PAGER=''
export AWS_RETRY_MODE=standard
export AWS_MAX_ATTEMPTS=2
fail() { printf '%s\n' "$*" >&2; exit 1; }
[[ $# -ge 1 && $# -le 2 ]] || fail 'Usage: deploy-canary.sh outputs.json [--check-only]'
outputs=$1
mode=${2:-}
[[ -z $mode || $mode == --check-only ]] || fail 'Unknown mode'
value() { jq -er --arg key "$1" '.[$key].value | select(type == "string" and length > 0)' "$outputs"; }
function_name=$(value resolve_function_name)
application=$(value codedeploy_application_name)
group=$(value codedeploy_deployment_group_name)
target=$(value resolve_published_version)
[[ $function_name =~ ^[a-z][a-z0-9-]+-resolve-link$ && $application == "${function_name%-link}" && $group == "$application-live" ]] || fail 'Unexpected function/application/group mapping'
[[ $target =~ ^[1-9][0-9]*$ ]] || fail 'Target must be a published numeric Lambda version'
poll=${CANARY_POLL_SECONDS:-5}
timeout=${CANARY_TIMEOUT_SECONDS:-1800}
[[ $poll =~ ^[1-9][0-9]*$ && $timeout =~ ^[1-9][0-9]*$ ]] || fail 'Invalid wait configuration'
(( poll <= 60 && timeout <= 3600 )) || fail 'Wait configuration exceeds bounds'
aws_cli() { aws "$@" --cli-connect-timeout 10 --cli-read-timeout 20 --output json; }
assert_idle() {
  local active
  active=$(aws_cli deploy list-deployments --application-name "$application" --deployment-group-name "$group" \
    --include-only-statuses Created Queued InProgress Ready Baking)
  jq -e '.deployments | type == "array" and length == 0' <<<"$active" >/dev/null || fail 'Active deployment exists; reconcile it before apply or rollout'
}
read_alias() { aws_cli lambda get-alias --function-name "$function_name" --name live; }
assert_alias() {
  local expected=$1 alias_json=$2
  jq -e --arg version "$expected" '.FunctionVersion == $version and ((.RoutingConfig.AdditionalVersionWeights // {}) | length == 0)' <<<"$alias_json" >/dev/null || fail 'Alias version/routing does not match the terminal deployment'
}
assert_idle
alias_json=$(read_alias)
current=$(jq -er '.FunctionVersion | select(test("^[1-9][0-9]*$"))' <<<"$alias_json")
assert_alias "$current" "$alias_json"
[[ $mode != --check-only ]] || { printf 'Canary writer is idle\n'; exit 0; }
[[ $current != "$target" ]] || { printf 'Same version %s; no deployment\n' "$target"; exit 0; }
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
jq -n --arg name "$function_name" --arg current "$current" --arg target "$target" \
  '{version:0.0,Resources:[{Resolve:{Type:"AWS::Lambda::Function",Properties:{Name:$name,Alias:"live",CurrentVersion:$current,TargetVersion:$target}}}]}' > "$work/appspec.json"
jq -n --arg app "$application" --arg group "$group" --rawfile spec "$work/appspec.json" \
  '{applicationName:$app,deploymentGroupName:$group,revision:{revisionType:"AppSpecContent",appSpecContent:{content:$spec}}}' > "$work/request.json"
# CreateDeployment has no idempotency token: a lost response requires reconciliation,
# never a blind retry or alias reset. Subsequent runs check active deployments first.
if ! response=$(AWS_MAX_ATTEMPTS=1 aws_cli deploy create-deployment --cli-input-json "file://$work/request.json"); then
  fail 'CreateDeployment outcome unknown; reconcile deployment history before retrying. Frontend publish blocked.'
fi
id=$(jq -er '.deploymentId | select(test("^d-[A-Za-z0-9]+$"))' <<<"$response")
printf 'Deployment %s: %s -> %s\n' "$id" "$current" "$target"
deadline=$((SECONDS + timeout))
wait_terminal() {
  local deployment=$1 state
  while (( SECONDS < deadline )); do
    info=$(aws_cli deploy get-deployment --deployment-id "$deployment")
    state=$(jq -er '.deploymentInfo.status' <<<"$info")
    case "$state" in
      Succeeded|Failed|Stopped) return 0 ;;
      Created|Queued|InProgress|Ready|Baking) sleep "$poll" ;;
      *) fail "Unknown deployment status: $state; reconcile $deployment" ;;
    esac
  done
  fail "Timed out waiting for $deployment; reconcile before retry. Frontend publish blocked."
}
wait_terminal "$id"
state=$(jq -r '.deploymentInfo.status' <<<"$info")
if [[ $state == Succeeded ]]; then
  assert_alias "$target" "$(read_alias)"
  assert_idle
  printf 'Canary succeeded: %s\n' "$id"
  exit 0
fi
# Rollback can be registered after the original deployment reaches terminal state.
# Keep a bounded wait for its ID; never infer rollback completion from parent status.
while (( SECONDS < deadline )); do
  rollback=$(jq -r '.deploymentInfo.rollbackInfo.rollbackDeploymentId // empty' <<<"$info")
  if [[ -n $rollback ]]; then
    [[ $rollback =~ ^d-[A-Za-z0-9]+$ ]] || fail 'Invalid rollback deployment ID'
    wait_terminal "$rollback"
    [[ $(jq -r '.deploymentInfo.status' <<<"$info") == Succeeded ]] || fail "Rollback $rollback failed; operator action required"
    assert_alias "$current" "$(read_alias)"
    assert_idle
    fail "Deployment $id failed; rollback $rollback completed. Frontend publish blocked."
  fi
  sleep "$poll"
  info=$(aws_cli deploy get-deployment --deployment-id "$id")
done
fail "Deployment $id failed; rollback completion unconfirmed. Frontend publish blocked."
