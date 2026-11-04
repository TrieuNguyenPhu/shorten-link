Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$scriptPath = Join-Path $PSScriptRoot 'aws-flow.ps1'
$tokens = $null
$errors = $null
[void][Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$tokens, [ref]$errors)
if ($errors.Count -gt 0) { throw ($errors | Out-String) }
[void][Management.Automation.Language.Parser]::ParseFile((Join-Path $PSScriptRoot 'pipeline.ps1'), [ref]$tokens, [ref]$errors)
if ($errors.Count -gt 0) { throw ($errors | Out-String) }
$temporary = Join-Path ([IO.Path]::GetTempPath()) ("url-shortener-e2e-" + [guid]::NewGuid().ToString('N') + '.json')
try {
    @{ http_api_endpoint = @{ value = 'https://example.org/' } } | ConvertTo-Json | Set-Content -LiteralPath $temporary
    $rejected = $false
    try { & $scriptPath -OutputsPath $temporary }
    catch { $rejected = $_.Exception.Message -eq 'The endpoint must be an AWS HTTP API HTTPS origin from Terraform outputs.' }
    if (-not $rejected) { throw 'The E2E script must reject a remote non-API-Gateway host before reading or sending tokens.' }
    @{ deployment_role_arn = @{ value = 'arn:aws:iam::111111111111:role/untrusted' }; github_subject = @{ value = 'repo:TrieuNguyenPhu/shorten-link:environment:dev' } } | ConvertTo-Json | Set-Content -LiteralPath $temporary
    $rejected = $false
    try { & $scriptPath -Scenario OIDC -OutputsPath $temporary }
    catch { $rejected = $_.Exception.Message -eq 'OIDC tests require bootstrap deployment_role_arn and exact github_subject outputs.' }
    if (-not $rejected) { throw 'The OIDC script must reject an untrusted role path before reading tokens or calling STS.' }
    @{ deployment_role_arn = @{ value = 'arn:aws:iam::111111111111:role/url-shortener/bootstrap/url-shortener-dev-github-deploy' }; github_subject = @{ value = 'repo:TrieuNguyenPhu/shorten-link:environment:dev' } } | ConvertTo-Json | Set-Content -LiteralPath $temporary
    $originalToken = $env:E2E_OIDC_ALLOWED_TOKEN
    try {
        $env:E2E_OIDC_ALLOWED_TOKEN = ''
        $rejected = $false
        try { & $scriptPath -Scenario OIDC -OutputsPath $temporary }
        catch { $rejected = $_.Exception.Message -eq 'Missing required environment input: E2E_OIDC_ALLOWED_TOKEN' }
        if (-not $rejected) { throw 'Valid OIDC config must fail closed before STS when no token is supplied.' }
    } finally { $env:E2E_OIDC_ALLOWED_TOKEN = $originalToken }
    Write-Host 'PowerShell parser and Core/OIDC preflight PASS; no AWS requests sent.'
} finally { Remove-Item -LiteralPath $temporary -Force }

. (Join-Path $PSScriptRoot 'deletion.ps1')
$script:readCount = 0
$result = Wait-DeletedLink -DelayMilliseconds 0 -ReadLink {
    $script:readCount++
    [pscustomobject]@{ Status = $(if ($script:readCount -lt 3) { 301 } else { 404 }) }
}
if ($result.Status -ne 404 -or $script:readCount -ne 3) { throw 'Deletion check must tolerate stale reads and stop at 404.' }
foreach ($status in @(301, 500)) {
    $script:readCount = 0
    $failed = $false
    try {
        Wait-DeletedLink -Attempts 3 -DelayMilliseconds 0 -ReadLink { $script:readCount++; [pscustomobject]@{ Status = $status } }
    } catch { $failed = $true }
    $expectedReads = if ($status -eq 301) { 3 } else { 1 }
    if (-not $failed -or $script:readCount -ne $expectedReads) { throw 'Deletion check must bound retries and surface non-consistency errors immediately.' }
}
Write-Host 'Deletion consistency, retry budget and error isolation PASS.'

$pipelineTemporary = Join-Path ([IO.Path]::GetTempPath()) ("url-shortener-pipeline-" + [guid]::NewGuid().ToString('N') + '.json')
$originalAccessToken = $env:E2E_ACCESS_TOKEN_A
try {
    @{ http_api_endpoint = @{ value = 'https://example.org/' } } | ConvertTo-Json | Set-Content -LiteralPath $pipelineTemporary
    $rejected = $false
    try { & $scriptPath -Scenario Pipeline -OutputsPath $pipelineTemporary }
    catch { $rejected = $_.Exception.Message -eq 'Pipeline requires matching AWS API, table, bus, queue, alias and ESM outputs.' }
    if (-not $rejected) { throw 'Pipeline must validate resource outputs before tokens or AWS calls.' }
    $config = @{
        http_api_endpoint = @{ value = 'https://api123.execute-api.ap-southeast-1.amazonaws.com' }
        table_name = @{ value = 'url-shortener' }
        table_arn = @{ value = 'arn:aws:dynamodb:ap-southeast-1:111111111111:table/url-shortener' }
        event_bus_name = @{ value = 'url-shortener-events' }
        event_bus_arn = @{ value = 'arn:aws:events:ap-southeast-1:111111111111:event-bus/url-shortener-events' }
        click_queue_arn = @{ value = 'arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickQueue' }
        click_queue_url = @{ value = 'https://sqs.ap-southeast-1.amazonaws.com/111111111111/url-shortener-dev-ClickQueue' }
        click_dlq_arn = @{ value = 'arn:aws:sqs:ap-southeast-1:111111111111:url-shortener-dev-ClickDLQ' }
        click_dlq_url = @{ value = 'https://sqs.ap-southeast-1.amazonaws.com/111111111111/url-shortener-dev-ClickDLQ' }
        click_rule_arn = @{ value = 'arn:aws:events:ap-southeast-1:111111111111:rule/url-shortener-events/url-shortener-dev-link-clicked' }
        click_event_source_mapping_id = @{ value = '12345678-1234-1234-1234-123456789012' }
        lambda_aliases = @{ value = @{ 'click-aggregator' = 'arn:aws:lambda:ap-southeast-1:111111111111:function:url-shortener-dev-click-aggregator:live' } }
        cognito_issuer = @{ value = 'https://cognito-idp.ap-southeast-1.amazonaws.com/ap-southeast-1_test' }
        cognito_client_id = @{ value = 'test-client' }
    }
    $config | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $pipelineTemporary
    $env:E2E_ACCESS_TOKEN_A = ''
    $rejected = $false
    try { & $scriptPath -Scenario Pipeline -OutputsPath $pipelineTemporary }
    catch { $rejected = $_.Exception.Message -eq 'Missing required environment input: E2E_ACCESS_TOKEN_A' }
    if (-not $rejected) { throw 'Valid Pipeline outputs must fail closed before AWS when the token is missing.' }
    $config.click_dlq_arn.value = 'arn:aws:sqs:us-east-1:222222222222:other-ClickDLQ'
    $config | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $pipelineTemporary
    $rejected = $false
    try { & $scriptPath -Scenario Pipeline -OutputsPath $pipelineTemporary }
    catch { $rejected = $_.Exception.Message -eq 'Pipeline requires matching AWS API, table, bus, queue, alias and ESM outputs.' }
    if (-not $rejected) { throw 'Pipeline must reject cross-account/region resource mismatches before AWS calls.' }
} finally {
    $env:E2E_ACCESS_TOKEN_A = $originalAccessToken
    Remove-Item -LiteralPath $pipelineTemporary -Force
}
Write-Host 'Pipeline output and missing-token preflight PASS; no AWS requests sent.'

. (Join-Path $PSScriptRoot 'pipeline.ps1')
foreach ($body in @('42','true','null','[1,2]','[{"id":"wanted"}]','{"id":["wanted"]}','{"id":"other"}','{}','{')) {
    if ($null -ne (Read-PipelineEnvelope -Body $body -EventID 'wanted')) { throw 'Unrelated/non-object DLQ JSON must stay unmatched for visibility restoration.' }
}
$matched = Read-PipelineEnvelope -Body '{"id":"wanted","detail":{"code":"Abc1234"}}' -EventID 'wanted'
if ($null -eq $matched -or $matched.detail.code -ne 'Abc1234') { throw 'The matching DLQ envelope must remain available for validation and cleanup.' }
Write-Host 'Pipeline DLQ JSON shape/matching regression PASS; no AWS requests sent.'
