# Explicit AWS integration scenario. CI runs parser/preflight regressions only.
function Get-RealtimeConfig([hashtable]$Outputs) {
    try {
        foreach ($key in @('http_api_endpoint','table_name','table_arn','websocket_api_id','websocket_endpoint','websocket_management_endpoint','websocket_connections_arn','moderation_state_machine_arn','moderation_role_arn','lambda_aliases','cognito_issuer','cognito_client_id')) {
            if (-not $Outputs.ContainsKey($key)) { throw 'Missing output' }
        }
        $match = [regex]::Match([string]$Outputs.moderation_state_machine_arn.value, '^arn:aws:states:(?<region>[a-z0-9-]+):(?<account>[0-9]{12}):stateMachine:(?<prefix>[a-z][a-z0-9-]{2,35})-moderation$')
        if (-not $match.Success) { throw 'Invalid machine' }
        $region=$match.Groups['region'].Value; $account=$match.Groups['account'].Value; $prefix=$match.Groups['prefix'].Value
        $base=[uri]$Outputs.http_api_endpoint.value; $table=[string]$Outputs.table_name.value; $api=[string]$Outputs.websocket_api_id.value
        $ws="wss://$api.execute-api.$region.amazonaws.com/prod"
        if ($base.Scheme -ne 'https' -or $base.Host -notmatch ('^[a-z0-9]+\.execute-api\.'+[regex]::Escape($region)+'\.amazonaws\.com$') -or $base.UserInfo -or $base.Query -or $base.Fragment -or $base.AbsolutePath -ne '/' -or $base.Port -ne 443 -or
            $table -notmatch '^[A-Za-z0-9_.-]{3,255}$' -or $Outputs.table_arn.value -ne "arn:aws:dynamodb:${region}:${account}:table/$table" -or $api -notmatch '^[a-z0-9]+$' -or
            $Outputs.websocket_endpoint.value -ne $ws -or $Outputs.websocket_management_endpoint.value -ne $ws.Replace('wss://','https://') -or
            $Outputs.websocket_connections_arn.value -ne "arn:aws:execute-api:${region}:${account}:${api}/prod/POST/@connections/*" -or
            $Outputs.moderation_role_arn.value -ne "arn:aws:iam::${account}:role/url-shortener/dev/$prefix-moderation-sfn" -or
            $Outputs.lambda_aliases.value['moderate-check'] -ne "arn:aws:lambda:${region}:${account}:function:$prefix-moderate-check:live" -or
            $Outputs.lambda_aliases.value['click-aggregator'] -ne "arn:aws:lambda:${region}:${account}:function:$prefix-click-aggregator:live" -or
            $Outputs.cognito_issuer.value -notmatch ('^https://cognito-idp\.'+[regex]::Escape($region)+'\.amazonaws\.com/'+[regex]::Escape($region)+'_[A-Za-z0-9]+$') -or [string]::IsNullOrWhiteSpace($Outputs.cognito_client_id.value)) { throw 'Mismatched outputs' }
        return @{Region=$region;Account=$account;Prefix=$prefix;Base=$base;Table=$table;API=$api;WS=$ws}
    } catch { throw 'Realtime requires matching AWS HTTP, WebSocket, table and moderation outputs.' }
}

function Invoke-Realtime([hashtable]$Outputs) {
    $cfg=Get-RealtimeConfig $Outputs
    $token=Require-Environment 'E2E_ACCESS_TOKEN_A'
    $claims=Read-Claims $token
    if ($claims['token_use'] -ne 'access' -or $claims['iss'] -ne $Outputs.cognito_issuer.value -or $claims['client_id'] -ne $Outputs.cognito_client_id.value -or 'openid' -notin ($claims['scope'] -split ' ') -or -not $claims['sub'] -or [long]$claims['exp'] -le [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()) { throw 'Realtime requires an unexpired Cognito access token with openid scope.' }
    if (-not (Get-Command aws -ErrorAction SilentlyContinue)) { throw 'Realtime requires AWS CLI v2 and operator credentials.' }
    function Invoke-RealtimeAWS([string[]]$Arguments,[hashtable]$InputObject=$null) {
        $temporary=$null
        try {
            if ($null -ne $InputObject) {
                $temporary=[IO.Path]::GetTempFileName()
                [IO.File]::WriteAllText($temporary,($InputObject|ConvertTo-Json -Depth 12 -Compress),[Text.UTF8Encoding]::new($false))
                $Arguments+=@('--cli-input-json',"file://$temporary")
            }
            $raw=& aws @Arguments --region $cfg.Region --output json --no-cli-pager --cli-connect-timeout 5 --cli-read-timeout 15 2>$null | Out-String
            if ($LASTEXITCODE -ne 0) { throw "AWS Realtime command failed: $($Arguments[0]) $($Arguments[1])" }
            if ([string]::IsNullOrWhiteSpace($raw)) { return @{} }
            return $raw|ConvertFrom-Json -AsHashtable
        } finally { if ($temporary) { Remove-Item -LiteralPath $temporary -Force } }
    }
    if ((Invoke-RealtimeAWS @('sts','get-caller-identity')).Account -ne $cfg.Account) { throw 'Operator credentials do not match the Terraform account.' }
    $machine=Invoke-RealtimeAWS @('stepfunctions','describe-state-machine','--state-machine-arn',[string]$Outputs.moderation_state_machine_arn.value)
    $definition=$machine.definition|ConvertFrom-Json -AsHashtable
    if ($machine.type -ne 'STANDARD' -or $machine.roleArn -ne $Outputs.moderation_role_arn.value -or $definition.States.CheckSafety.Resource -ne $Outputs.lambda_aliases.value['moderate-check'] -or $definition.States.Activate.Parameters.TableName -ne $cfg.Table) { throw 'Unexpected deployed moderation definition.' }
    # GetApi is account-scoped: verify both API IDs before sending a bearer token or opening sockets.
    $httpAPI=Invoke-RealtimeAWS @('apigatewayv2','get-api','--api-id',$cfg.Base.Host.Split('.')[0])
    $wsAPI=Invoke-RealtimeAWS @('apigatewayv2','get-api','--api-id',$cfg.API)
    if ($httpAPI.ProtocolType -ne 'HTTP' -or $httpAPI.ApiEndpoint.TrimEnd('/') -ne $cfg.Base.AbsoluteUri.TrimEnd('/') -or $wsAPI.ProtocolType -ne 'WEBSOCKET' -or "$($wsAPI.ApiEndpoint)/prod" -ne $cfg.WS) { throw 'API IDs/endpoints do not match operator-owned APIs.' }
    $stage=Invoke-RealtimeAWS @('apigatewayv2','get-stage','--api-id',$cfg.API,'--stage-name','prod')
    $aggregator=Invoke-RealtimeAWS @('lambda','get-function-configuration','--function-name',[string]$Outputs.lambda_aliases.value['click-aggregator'])
    if (-not $stage.AutoDeploy -or $aggregator.Environment.Variables.WS_ENDPOINT -ne $Outputs.websocket_management_endpoint.value) { throw 'Unexpected deployed WebSocket stage/aggregator configuration.' }
    $transport=[Net.Http.HttpClientHandler]::new(); $transport.AllowAutoRedirect=$false
    $client=[Net.Http.HttpClient]::new($transport); $client.Timeout=[TimeSpan]::FromSeconds(10)
    $codes=[Collections.Generic.List[string]]::new(); $running=[Collections.Generic.HashSet[string]]::new()
    $sockets=[Collections.Generic.List[Net.WebSockets.ClientWebSocket]]::new()
    function Invoke-RealtimeApi([string]$Method,[string]$Path,[string]$Body='') {
        $request=[Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::new($Method),"$($cfg.Base.AbsoluteUri.TrimEnd('/'))$Path")
        try {
            if ($Method -ne 'GET') { $request.Headers.Authorization=[Net.Http.Headers.AuthenticationHeaderValue]::new('Bearer',$token) }
            if ($Body) { $request.Content=[Net.Http.StringContent]::new($Body,[Text.Encoding]::UTF8,'application/json') }
            $response=$client.SendAsync($request).GetAwaiter().GetResult()
            try { return @{Status=[int]$response.StatusCode;Body=$response.Content.ReadAsStringAsync().GetAwaiter().GetResult();Location=$response.Headers.Location} }
            finally { $response.Dispose() }
        } finally { $request.Dispose() }
    }
    function Read-RealtimeMeta([string]$Code) {
        $out=Invoke-RealtimeAWS @('dynamodb','get-item') @{TableName=$cfg.Table;ConsistentRead=$true;Key=@{PK=@{S="LINK#$Code"};SK=@{S='META'}}}
        if (-not $out.ContainsKey('Item')) { throw 'Test link META disappeared; do not run moderation concurrently with deletion.' }
        return $out.Item
    }
    function Start-RealtimeModeration([string]$Code,[string]$Target,[string]$Expected) {
        # Re-check the complete META immediately before execution to avoid known upserts.
        $meta=Read-RealtimeMeta $Code
        if ($meta.ownerId.S -ne $claims['sub'] -or -not $meta.createdAt.S -or -not $meta.ContainsKey('clicks') -or ($Target -and $meta.target.S -ne $Target)) { throw 'Invalid test META; refusing moderation.' }
        $started=Invoke-RealtimeAWS @('stepfunctions','start-execution') @{stateMachineArn=$Outputs.moderation_state_machine_arn.value;input=(@{code=$Code;target=$Target}|ConvertTo-Json -Compress)}
        $arn=[string]$started.executionArn; [void]$running.Add($arn)
        for ($attempt=0;$attempt -lt 40;$attempt++) {
            $execution=Invoke-RealtimeAWS @('stepfunctions','describe-execution','--execution-arn',$arn)
            if ($execution.status -ne 'RUNNING') {
                [void]$running.Remove($arn)
                if ($execution.status -ne $Expected) { throw 'Unexpected moderation execution outcome.' }
                return @{Execution=$execution;History=(Invoke-RealtimeAWS @('stepfunctions','get-execution-history','--execution-arn',$arn))}
            }
            Start-Sleep -Milliseconds 500
        }
        throw 'Moderation did not finish within the bounded wait.'
    }
    function Get-RealtimeConnections([string]$Code) {
        $out=Invoke-RealtimeAWS @('dynamodb','query') @{TableName=$cfg.Table;IndexName='GSI1';KeyConditionExpression='GSI1PK = :code';ExpressionAttributeValues=@{':code'=@{S="WSCODE#$Code"}}}
        return @($out.Items)
    }
    try {
        $links=@()
        foreach ($kind in @('docs','evil')) {
            $target="https://example.org/realtime/$kind/$([guid]::NewGuid().ToString('N'))"
            $created=Invoke-RealtimeApi POST /links (@{url=$target}|ConvertTo-Json -Compress)
            if ($created.Status -ne 201) { throw 'Realtime create failed.' }
            $link=$created.Body|ConvertFrom-Json -AsHashtable
            if ($link.code -notmatch '^[A-Za-z0-9]{7}$' -or $link.target -ne $target -or $link.shortUrl -ne "$($cfg.Base.AbsoluteUri.TrimEnd('/'))/$($link.code)") { throw 'Create returned an unverified test code.' }
            $codes.Add($link.code); $links+=,$link
            $meta=Read-RealtimeMeta $link.code
            if ($meta.target.S -ne $target) { throw 'Test META target differs from create.' }
            $result=Start-RealtimeModeration $link.code $target SUCCEEDED
            $expectedStatus=if ($kind -eq 'docs') {'active'} else {'rejected'}
            $after=Read-RealtimeMeta $link.code
            if ($after.status.S -ne $expectedStatus -or $after.target.S -ne $target -or $after.ownerId.S -ne $meta.ownerId.S -or $after.createdAt.S -ne $meta.createdAt.S -or $after.clicks.N -ne $meta.clicks.N) { throw 'Moderation changed unexpected metadata.' }
        }
        $safe=$links[0]; $failed=Start-RealtimeModeration $safe.code '' FAILED
        $scheduled=@($failed.History.events|Where-Object {$_.type -eq 'LambdaFunctionScheduled'})
        $failStates=@($failed.History.events|Where-Object {$_.type -eq 'FailStateEntered' -and $_.stateEnteredEventDetails.name -eq 'MarkError'})
        if ($failed.Execution.error -ne 'ModerationFailed' -or $scheduled.Count -ne 4 -or $failStates.Count -ne 1 -or (Read-RealtimeMeta $safe.code).status.S -ne 'active') { throw 'Retry/Catch/Fail did not preserve the contract.' }
        foreach ($index in 1..2) {
            $socket=[Net.WebSockets.ClientWebSocket]::new(); $sockets.Add($socket)
            $budget=[Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(10))
            try { $socket.ConnectAsync([uri]"$($cfg.WS)?code=$($safe.code)",$budget.Token).GetAwaiter().GetResult() } finally { $budget.Dispose() }
        }
        $ready=$false
        for ($attempt=0;$attempt -lt 20;$attempt++) {
            $connections=@(Get-RealtimeConnections $safe.code)
            if ($connections.Count -eq 2) { $ready=$true; break }
            Start-Sleep -Milliseconds 250
        }
        if (-not $ready) { throw 'Two subscriptions did not appear within the bounded GSI wait.' }
        foreach ($connection in $connections) {
            $remaining=[long]$connection.ttl.N-[DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
            if ($connection.code.S -ne $safe.code -or $remaining -lt 6900 -or $remaining -gt 7200) { throw 'Unexpected subscription code/TTL.' }
        }
        $resolved=Invoke-RealtimeApi GET "/$($safe.code)"
        if ($resolved.Status -ne 301 -or $resolved.Location.AbsoluteUri -ne $safe.target) { throw 'Realtime redirect failed.' }
        foreach ($socket in $sockets) {
            $buffer=[byte[]]::new(4096); $budget=[Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(15))
            try { $message=$socket.ReceiveAsync([ArraySegment[byte]]::new($buffer),$budget.Token).GetAwaiter().GetResult() } finally { $budget.Dispose() }
            if ($message.MessageType -ne [Net.WebSockets.WebSocketMessageType]::Text -or -not $message.EndOfMessage) { throw 'Unexpected WebSocket frame.' }
            $payload=[Text.Encoding]::UTF8.GetString($buffer,0,$message.Count)|ConvertFrom-Json -AsHashtable
            if ($payload.code -ne $safe.code -or [long]$payload.clicks -ne 1 -or [long](Read-RealtimeMeta $safe.code).clicks.N -ne 1) { throw 'Realtime payload does not match the committed count.' }
        }
    } finally {
        foreach ($socket in $sockets) {
            try {
                if ($socket.State -eq [Net.WebSockets.WebSocketState]::Open) {
                    $budget=[Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(5))
                    try { $socket.CloseAsync([Net.WebSockets.WebSocketCloseStatus]::NormalClosure,'done',$budget.Token).GetAwaiter().GetResult() } finally { $budget.Dispose() }
                }
            } catch { Write-Warning 'WebSocket close did not complete; stale connections expire through TTL/410 cleanup.' }
            finally { $socket.Dispose() }
        }
        $safeToDelete=$true
        foreach ($arn in @($running)) {
            try {
                [void](Invoke-RealtimeAWS @('stepfunctions','stop-execution','--execution-arn',$arn,'--error','E2ECleanup'))
                $terminal=$false
                for ($attempt=0;$attempt -lt 20;$attempt++) {
                    if ((Invoke-RealtimeAWS @('stepfunctions','describe-execution','--execution-arn',$arn)).status -ne 'RUNNING') {$terminal=$true;break}
                    Start-Sleep -Milliseconds 250
                }
                if (-not $terminal) {$safeToDelete=$false}
            } catch {$safeToDelete=$false}
        }
        $cleanupFailed=$false
        if ($safeToDelete) {
            foreach ($code in $codes) {
                try {if ((Invoke-RealtimeApi DELETE "/links/$code").Status -ne 200) {$cleanupFailed=$true}}
                catch {$cleanupFailed=$true}
            }
        } else {Write-Warning "Executions may still run; retain test links until terminal: $($codes -join ',')"; $cleanupFailed=$true}
        $client.Dispose(); $transport.Dispose()
        if ($cleanupFailed) {throw 'Realtime cleanup incomplete; inspect test links and executions before retrying.'}
    }
    Write-Host 'Realtime AWS E2E PASS: safe/rejected, four retry attempts with Fail, and two real WebSocket pushes.'
}
