# Wait only for stale reads after an acknowledged delete; other failures stay visible.
function Wait-DeletedLink {
    param(
        [Parameter(Mandatory)][scriptblock]$ReadLink,
        [ValidateRange(1, 20)][int]$Attempts = 10,
        [ValidateRange(0, 1000)][int]$DelayMilliseconds = 250
    )
    for ($attempt = 0; $attempt -lt $Attempts; $attempt++) {
        $result = & $ReadLink
        if ($result.Status -eq 404) { return $result }
        if ($result.Status -ne 301) { throw "Unexpected status while checking deletion: $($result.Status)." }
        if ($attempt + 1 -lt $Attempts) { Start-Sleep -Milliseconds $DelayMilliseconds }
    }
    throw 'Deleted link still resolves after the bounded consistency wait.'
}
