[CmdletBinding()]
param(
    [ValidateSet('Core', 'OIDC', 'Pipeline')]
    [string]$Scenario = 'Core',
    [Parameter(Mandatory)]
    [string]$OutputsPath
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'deletion.ps1')

# Tokens are read from process environment, never arguments or evidence output.
function Read-Claims([string]$Token) {
    try {
        $segment = $Token.Split('.')[1].Replace('-', '+').Replace('_', '/')
        $segment = $segment.PadRight($segment.Length + ((4 - $segment.Length % 4) % 4), '=')
        return [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($segment)) | ConvertFrom-Json -AsHashtable
    } catch { throw 'Invalid JWT format in E2E token input.' }
}
function Require-Environment([string]$Name) {
    $value = [Environment]::GetEnvironmentVariable($Name)
    if ([string]::IsNullOrWhiteSpace($value)) { throw "Missing required environment input: $Name" }
    return $value
}

$outputs = Get-Content -LiteralPath $OutputsPath -Raw | ConvertFrom-Json -AsHashtable
if ($Scenario -eq 'Pipeline') {
    . (Join-Path $PSScriptRoot 'pipeline.ps1')
    Invoke-Pipeline $outputs
    return
}
if ($Scenario -eq 'OIDC') {
    $roleArn = $outputs.deployment_role_arn.value
    $subject = $outputs.github_subject.value
    if ($roleArn -notmatch '^arn:aws:iam::[0-9]{12}:role/url-shortener/bootstrap/[a-z0-9-]+-github-deploy$' -or $subject -notmatch '^repo:([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+):(?:ref:refs/heads/main|environment:[A-Za-z0-9_-]+)$') {
        throw 'OIDC tests require bootstrap deployment_role_arn and exact github_subject outputs.'
    }
    $repository = $Matches[1]
    $cases = @(
        @{ Name = 'allowed subject'; Variable = 'E2E_OIDC_ALLOWED_TOKEN'; Allowed = $true },
        @{ Name = 'wrong audience'; Variable = 'E2E_OIDC_WRONG_AUDIENCE_TOKEN'; Allowed = $false },
        @{ Name = 'wrong subject'; Variable = 'E2E_OIDC_WRONG_SUBJECT_TOKEN'; Allowed = $false },
        @{ Name = 'wrong ref'; Variable = 'E2E_OIDC_WRONG_REF_TOKEN'; Allowed = $false },
        @{ Name = 'fork repository'; Variable = 'E2E_OIDC_FORK_TOKEN'; Allowed = $false }
    )
    # Validate every input before opening an STS connection. Tokens must be real GitHub-signed JWTs.
    foreach ($case in $cases) {
        $case.Token = Require-Environment $case.Variable
        $claims = Read-Claims $case.Token
        if ($claims['iss'] -ne 'https://token.actions.githubusercontent.com' -or [long]$claims['exp'] -le [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()) {
            throw 'OIDC tests require unexpired GitHub Actions tokens.'
        }
        $audienceMatches = 'sts.amazonaws.com' -in @($claims['aud'])
        if ($case.Name -eq 'wrong audience') {
            if ($audienceMatches -or $claims['sub'] -ne $subject) { throw 'Wrong-audience token must keep the allowed subject and use a different audience.' }
        } elseif (-not $audienceMatches) { throw 'All other OIDC test tokens must use the STS audience.' }
        if ($case.Allowed -and $claims['sub'] -ne $subject) { throw 'Allowed token does not match the configured subject.' }
        if (-not $case.Allowed -and $case.Name -ne 'wrong audience' -and $claims['sub'] -eq $subject) { throw 'Negative OIDC token matches the allowed subject.' }
        if ($case.Name -eq 'wrong ref' -and (-not $claims['sub'].StartsWith("repo:$($repository):ref:refs/heads/") -or $claims['sub'] -eq "repo:$($repository):ref:refs/heads/main")) {
            throw 'Wrong-ref token must use another branch subject, without a protected environment.'
        }
        if ($case.Name -eq 'wrong subject' -and -not $claims['sub'].StartsWith("repo:$($repository):")) { throw 'Wrong-subject token must use the same repository with a different environment or PR subject.' }
        if ($case.Name -eq 'fork repository'  -and $claims['sub'].StartsWith("repo:$($repository):")) { throw 'Fork token must originate from a different repository.' }
    }
    $handler = [Net.Http.HttpClientHandler]::new()
    $handler.AllowAutoRedirect = $false
    $sts = [Net.Http.HttpClient]::new($handler)
    $sts.Timeout = [TimeSpan]::FromSeconds(15)
    $results = [Collections.Generic.List[object]]::new()
    try {
        foreach ($case in $cases) {
            $fields = [Collections.Generic.Dictionary[string,string]]::new()
            $fields.Add('Action', 'AssumeRoleWithWebIdentity')
            $fields.Add('Version', '2011-06-15')
            $fields.Add('RoleArn', $roleArn)
            $fields.Add('RoleSessionName', "e2e-$([guid]::NewGuid().ToString('N'))")
            $fields.Add('DurationSeconds', '900')
            $fields.Add('WebIdentityToken', $case.Token)
            $content = [Net.Http.FormUrlEncodedContent]::new($fields)
            try {
                try { $response = $sts.PostAsync('https://sts.amazonaws.com/', $content).GetAwaiter().GetResult() }
                catch { throw "STS request failed for check: $($case.Name)" }
                try {
                    $status = [int]$response.StatusCode
                    if ($case.Allowed) {
                        if ($status -ne 200) { throw "Allowed OIDC identity was rejected: status $status." }
                        # Do not deserialize, output, persist or use returned session credentials.
                    } else {
                        if ($status -notin @(400, 403)) { throw "Negative OIDC check had unexpected status $status." }
                        $errorDocument = [xml]$response.Content.ReadAsStringAsync().GetAwaiter().GetResult()
                        $errorCode = $errorDocument.ErrorResponse.Error.Code
                        $expectedCode = if ($case.Name -eq 'wrong audience') { 'InvalidIdentityToken' } else { 'AccessDenied' }
                        if ($errorCode -ne $expectedCode) { throw "Negative OIDC check returned $errorCode instead of $expectedCode." }
                    }
                    $results.Add([pscustomobject]@{ Check = $case.Name; Status = $status })
                } finally { $response.Dispose() }
            } finally { $content.Dispose() }
        }
    } finally { $sts.Dispose(); $handler.Dispose() }
    $results | ConvertTo-Json
    Write-Host 'OIDC E2E PASS; no returned credentials were used or persisted.'
    return
}
$base = [uri]$outputs.http_api_endpoint.value
if ($base.Scheme -ne 'https' -or $base.Host -notmatch '^[a-z0-9]+\.execute-api\.[a-z0-9-]+\.amazonaws\.com$' -or $base.UserInfo -or $base.Query -or $base.Fragment -or $base.AbsolutePath -ne '/') {
    throw 'The endpoint must be an AWS HTTP API HTTPS origin from Terraform outputs.'
}
$issuer = $outputs.cognito_issuer.value
$clientId = $outputs.cognito_client_id.value
$tokenA = Require-Environment 'E2E_ACCESS_TOKEN_A'
$tokenB = Require-Environment 'E2E_ACCESS_TOKEN_B'
$idToken = Require-Environment 'E2E_ID_TOKEN_A'
$googleToken = Require-Environment 'E2E_GOOGLE_ID_TOKEN'
$claimsA = Read-Claims $tokenA
$claimsB = Read-Claims $tokenB
foreach ($claims in @($claimsA, $claimsB)) {
    if ($claims['token_use'] -ne 'access' -or $claims['iss'] -ne $issuer -or $claims['client_id'] -ne $clientId -or 'openid' -notin ($claims['scope'] -split ' ') -or -not $claims['sub'] -or [long]$claims['exp'] -le [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()) {
        throw 'Use unexpired Cognito access tokens with openid scope from the configured Google federation client.'
    }
}
if ($claimsA['sub'] -eq $claimsB['sub']) { throw 'Use two distinct Google test accounts.' }
$idClaims = Read-Claims $idToken
$googleClaims = Read-Claims $googleToken
if ($idClaims['token_use'] -ne 'id' -or $idClaims['iss'] -ne $issuer -or $idClaims['aud'] -ne $clientId -or $idClaims['sub'] -ne $claimsA['sub'] -or [long]$idClaims['exp'] -le [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() -or $googleClaims['iss'] -notin @('https://accounts.google.com', 'accounts.google.com') -or [long]$googleClaims['exp'] -le [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()) {
    throw 'Negative checks require a Cognito ID token and a Google ID token, respectively.'
}

$transport = [Net.Http.HttpClientHandler]::new()
$transport.AllowAutoRedirect = $false
$client = [Net.Http.HttpClient]::new($transport)
$client.Timeout = [TimeSpan]::FromSeconds(10)
$evidence = [Collections.Generic.List[object]]::new()
function Invoke-Api([string]$Method, [string]$Path, [string]$Token, [string]$Body = '') {
    $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::new($Method), "$($base.AbsoluteUri.TrimEnd('/'))$Path")
    try {
        if ($Token) { $request.Headers.Authorization = [Net.Http.Headers.AuthenticationHeaderValue]::new('Bearer', $Token) }
        if ($Body) { $request.Content = [Net.Http.StringContent]::new($Body, [Text.Encoding]::UTF8, 'application/json') }
        try { $response = $client.SendAsync($request).GetAwaiter().GetResult() }
        catch { throw "HTTP request failed: $Method $Path" }
        try {
            return [pscustomobject]@{ Status = [int]$response.StatusCode; Body = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult(); Location = $response.Headers.Location }
        } finally { $response.Dispose() }
    } finally { $request.Dispose() }
}
function Assert-Status([string]$Check, $Result, [int[]]$Expected) {
    $evidence.Add([pscustomobject]@{ Check = $Check; Status = $Result.Status })
    if ($Result.Status -notin $Expected) { throw "$Check failed: status $($Result.Status), expected $($Expected -join '/')." }
}

$code = $null
try {
    Assert-Status 'missing JWT' (Invoke-Api GET /links '') @(401)
    Assert-Status 'Cognito ID token rejected' (Invoke-Api GET /links $idToken) @(401, 403)
    Assert-Status 'Google token rejected' (Invoke-Api GET /links $googleToken) @(401, 403)
    foreach ($body in @('null', '{"url":12}', '{"url":"javascript:alert(1)"}')) {
        Assert-Status 'invalid create body' (Invoke-Api POST /links $tokenA $body) @(400)
    }
    $target = "https://example.org/e2e/$([guid]::NewGuid().ToString('N'))"
    $created = Invoke-Api POST /links $tokenA (@{ url = $target } | ConvertTo-Json -Compress)
    Assert-Status 'A creates link' $created @(201)
    $link = $created.Body | ConvertFrom-Json -AsHashtable
    # Capture only a generated code; never clean up a pre-existing link.
    if ($link.code -notmatch '^[A-Za-z0-9]{7}$') { throw 'Create returned an invalid code.' }
    if ($link.target -ne $target -or $link.shortUrl -ne "$($base.AbsoluteUri.TrimEnd('/'))/$($link.code)") { throw 'Create response does not match the API redirect domain.' }
    $code = $link.code
    $listed = $false
    for ($attempt = 0; $attempt -lt 10; $attempt++) {
        $listA = Invoke-Api GET /links $tokenA
        Assert-Status 'A lists links' $listA @(200)
        $linksA = ($listA.Body | ConvertFrom-Json -AsHashtable).links
        if ($code -in @($linksA | ForEach-Object { $_.code })) { $listed = $true; break }
        Start-Sleep -Milliseconds 250
    }
    if (-not $listed) { throw 'Created link did not appear in owner GSI within the bounded wait.' }
    $listB = Invoke-Api GET /links $tokenB
    Assert-Status 'B lists links' $listB @(200)
    if ($code -in @(($listB.Body | ConvertFrom-Json -AsHashtable).links | ForEach-Object { $_.code })) { throw 'Tenant isolation failed.' }
    $denied = Invoke-Api DELETE "/links/$code" $tokenB
    $missing = Invoke-Api DELETE "/links/$([guid]::NewGuid().ToString('N'))" $tokenB
    Assert-Status 'B cannot delete A link' $denied @(404)
    Assert-Status 'missing delete' $missing @(404)
    if ($denied.Body -ne $missing.Body) { throw 'Delete exposes an ownership oracle.' }
    $resolved = Invoke-Api GET "/$code" ''
    Assert-Status 'public redirect' $resolved @(301)
    if ($resolved.Location.AbsoluteUri -ne $target -or $resolved.Body -ne '') { throw 'Invalid 301 Location or body.' }
} finally {
    try {
        if ($code) {
            Assert-Status 'cleanup created link' (Invoke-Api DELETE "/links/$code" $tokenA) @(200)
            Assert-Status 'deleted link no longer resolves' (Wait-DeletedLink -ReadLink { Invoke-Api GET "/$code" '' }) @(404)
        }
    } finally { $client.Dispose(); $transport.Dispose() }
}
$evidence | ConvertTo-Json
Write-Host 'Core E2E PASS; created link cleaned up. Event delivery and aggregation require Phase 3.'
