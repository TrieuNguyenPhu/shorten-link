Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
# Exercise the real nested start function and finally block without running the
# cloud scenario. Only AWS/HTTP boundaries, clock delay and disposable clients are fixtures.
$tokens = $null; $parseErrors = $null
$ast = [Management.Automation.Language.Parser]::ParseFile((Join-Path $PSScriptRoot 'realtime.ps1'), [ref]$tokens, [ref]$parseErrors)
if ($parseErrors.Count) { throw ($parseErrors | Out-String) }
$start = $ast.FindAll({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Start-RealtimeModeration' }, $true)
Invoke-Expression $start[0].Extent.Text
$mainTry = $ast.FindAll({ param($node) $node -is [Management.Automation.Language.TryStatementAst] -and $null -ne $node.Finally -and $node.Finally.Extent.Text.Contains('$safeToDelete') }, $true)
$cleanupText = $mainTry[0].Finally.Extent.Text
$cleanup = [scriptblock]::Create($cleanupText.Substring(1, $cleanupText.Length - 2))
$claims = @{ sub = 'alice' }
$Outputs = @{ moderation_state_machine_arn = @{ value = 'arn:aws:states:ap-southeast-1:111111111111:stateMachine:test' } }
function Read-RealtimeMeta([string]$Code) { return @{ ownerId=@{S='alice'};createdAt=@{S='2026-11-04T00:00:00.000Z'};target=@{S='https://example.org'};clicks=@{N='0'} } }
function Start-Sleep([int]$Milliseconds) {}
function Invoke-RealtimeAWS([string[]]$Arguments, [hashtable]$InputObject = $null) {
    switch ($Arguments[1]) {
        'start-execution' {
            $script:startInput = $InputObject
            if ($running.Count -ne 1) { throw 'Execution must be registered before the request leaves the process.' }
            $script:arn = @($running)[0]
            $script:calls.Add('start')
            if ($script:mode -notin @('success','poll-error','pending')) { throw 'Response lost after server accepted execution.' }
            return @{ executionArn = $script:arn }
        }
        'describe-execution' {
            if ($Arguments[-1] -ne $script:arn) { throw 'Describe must reconcile the pre-registered execution ARN.' }
            $script:calls.Add('describe')
            if ($script:mode -in @('unknown','poll-error')) { throw 'Cannot confirm execution state.' }
            $status = if ($script:mode -eq 'pending') { 'PENDING_REDRIVE' } elseif ($script:mode -eq 'running' -or ($script:mode -in @('lost-running','lost-stop') -and -not $script:stopped)) { 'RUNNING' } elseif ($script:stopped) { 'ABORTED' } else { 'SUCCEEDED' }
            if ($status -in @('SUCCEEDED','FAILED','TIMED_OUT','ABORTED')) { $script:terminalObserved = $true }
            return @{ status = $status }
        }
        'stop-execution' {
            if ($Arguments[3] -ne $script:arn) { throw 'Stop must use the pre-registered ARN.' }
            $script:calls.Add('stop')
            if ($script:mode -eq 'running') { throw 'Stop denied; execution is still running.' }
            $script:stopped = $true
            if ($script:mode -eq 'lost-stop') { throw 'Stop response lost after server aborted execution.' }
            return @{}
        }
        'get-execution-history' { return @{ events=@() } }
        default { throw "Unexpected AWS operation: $Arguments" }
    }
}
function Invoke-RealtimeApi([string]$Method, [string]$Path) {
    if ($Method -ne 'DELETE' -or $Path -ne '/links/Abc1234') { throw 'Unexpected HTTP operation.' }
    $script:calls.Add('delete')
    if (-not $script:terminalObserved) { throw 'DELETE attempted before terminal execution was confirmed.' }
    return @{ Status=200 }
}
foreach ($case in @(
    @{Mode='lost-running';Delete=$true;Stop=1},
    @{Mode='lost-completed';Delete=$true;Stop=0},
    @{Mode='unknown';Delete=$false;Stop=0},
    @{Mode='running';Delete=$false;Stop=1},
    @{Mode='pending';Delete=$false;Stop=0},
    @{Mode='poll-error';Delete=$false;Stop=0},
    @{Mode='lost-stop';Delete=$true;Stop=1},
    @{Mode='success';Delete=$true;Stop=0}
)) {
    $script:mode = $case.Mode
    $script:stopped = $false; $script:terminalObserved = $false; $script:startInput = @{}; $script:arn = ''
    $script:calls = [Collections.Generic.List[string]]::new()
    $running = [Collections.Generic.HashSet[string]]::new()
    $codes = [Collections.Generic.List[string]]::new(); $codes.Add('Abc1234')
    $sockets = @()
    $client = [pscustomobject]@{}; $client | Add-Member ScriptMethod Dispose {}
    $transport = [pscustomobject]@{}; $transport | Add-Member ScriptMethod Dispose {}
    $startFailed = $false; $cleanupFailed = $false
    try { Start-RealtimeModeration 'Abc1234' 'https://example.org' 'SUCCEEDED' | Out-Null }
    catch { $startFailed = $true }
    try { & $cleanup 3>$null }
    catch { $cleanupFailed = $true }
    if ($script:mode -eq 'success' -and $startFailed) { throw 'Successful moderation unexpectedly failed.' }
    if ($script:mode -ne 'success' -and -not $startFailed) { throw 'Boundary failure was swallowed.' }
    if (-not $script:startInput.ContainsKey('name') -or $script:startInput.name -notmatch '^e2e-Abc1234-[0-9a-f]{32}$' -or $script:arn -ne "arn:aws:states:ap-southeast-1:111111111111:execution:test:$($script:startInput.name)" -or ($script:startInput.input | ConvertFrom-Json).target -ne 'https://example.org') { throw 'StartExecution must use a known unique name/ARN and preserve its input.' }
    if (('delete' -in $script:calls) -ne $case.Delete -or $cleanupFailed -eq $case.Delete -or @($script:calls | Where-Object { $_ -eq 'stop' }).Count -ne $case.Stop) { throw "Lifecycle cleanup regression: $($case.Mode); calls=$($script:calls -join ',')" }
    if (-not $case.Delete -and $running.Count -ne 1) { throw 'Unresolved execution must remain tracked.' }
    if ($case.Delete -and $running.Count -ne 0) { throw 'Confirmed terminal execution must be removed from tracking.' }
    Write-Host "Realtime lifecycle $($case.Mode) PASS"
}
