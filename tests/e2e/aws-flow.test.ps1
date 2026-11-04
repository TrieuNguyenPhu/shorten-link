Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$scriptPath = Join-Path $PSScriptRoot 'aws-flow.ps1'
$tokens = $null
$errors = $null
[void][Management.Automation.Language.Parser]::ParseFile($scriptPath, [ref]$tokens, [ref]$errors)
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
