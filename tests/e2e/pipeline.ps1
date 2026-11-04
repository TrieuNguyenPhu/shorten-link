# A DLQ can contain unrelated JSON scalars/arrays. Match only an object with a string ID.
function Read-PipelineEnvelope([string]$Body, [string]$EventID) {
    try { $decoded = ConvertFrom-Json -InputObject $Body -AsHashtable -NoEnumerate }
    catch { return $null }
    if ($decoded -is [System.Collections.IDictionary] -and $decoded.ContainsKey('id') -and $decoded.id -is [string] -and $decoded.id -eq $EventID) { return $decoded }
    return $null
}

# Explicit AWS integration scenario. Only the preflight is executed in CI.
function Invoke-Pipeline([hashtable]$Outputs) {
    try {
        foreach ($key in @('http_api_endpoint','table_name','table_arn','event_bus_name','event_bus_arn','click_queue_arn','click_queue_url','click_dlq_arn','click_dlq_url','click_rule_arn','click_event_source_mapping_id','lambda_aliases','cognito_issuer','cognito_client_id')) {
            if (-not $Outputs.ContainsKey($key)) { throw 'Missing output' }
        }
        $match = [regex]::Match([string]$Outputs.click_queue_arn.value, '^arn:aws:sqs:(?<region>[a-z0-9-]+):(?<account>[0-9]{12}):(?<prefix>[a-z][a-z0-9-]{2,35})-ClickQueue$')
        if (-not $match.Success) { throw 'Invalid queue ARN' }
        $region = $match.Groups['region'].Value
        $account = $match.Groups['account'].Value
        $prefix = $match.Groups['prefix'].Value
        $base = [uri]$Outputs.http_api_endpoint.value
        $table = [string]$Outputs.table_name.value
        $bus = [string]$Outputs.event_bus_name.value
        $queueURL = [string]$Outputs.click_queue_url.value
        $dlqURL = [string]$Outputs.click_dlq_url.value
        $alias = [string]$Outputs.lambda_aliases.value['click-aggregator']
        if ($base.Scheme -ne 'https' -or $base.Host -notmatch ('^[a-z0-9]+\.execute-api\.' + [regex]::Escape($region) + '\.amazonaws\.com$') -or $base.UserInfo -or $base.Query -or $base.Fragment -or $base.AbsolutePath -ne '/' -or
            $table -notmatch '^[A-Za-z0-9_.-]{3,255}$' -or $Outputs.table_arn.value -ne "arn:aws:dynamodb:${region}:${account}:table/$table" -or $bus -notmatch '^[A-Za-z0-9._-]{1,256}$' -or $bus -eq 'default' -or
            $Outputs.event_bus_arn.value -ne "arn:aws:events:${region}:${account}:event-bus/$bus" -or
            $queueURL -ne "https://sqs.$region.amazonaws.com/$account/$prefix-ClickQueue" -or $dlqURL -ne "https://sqs.$region.amazonaws.com/$account/$prefix-ClickDLQ" -or
            $Outputs.click_dlq_arn.value -ne "arn:aws:sqs:${region}:${account}:$prefix-ClickDLQ" -or
            $Outputs.click_rule_arn.value -ne "arn:aws:events:${region}:${account}:rule/$bus/$prefix-link-clicked" -or
            $alias -ne "arn:aws:lambda:${region}:${account}:function:$prefix-click-aggregator:live" -or
            $Outputs.click_event_source_mapping_id.value -notmatch '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$') { throw 'Mismatched outputs' }
    } catch { throw 'Pipeline requires matching AWS API, table, bus, queue, alias and ESM outputs.' }
    $token = Require-Environment 'E2E_ACCESS_TOKEN_A'
    $claims = Read-Claims $token
    if ($claims['token_use'] -ne 'access' -or $claims['iss'] -ne $Outputs.cognito_issuer.value -or $claims['client_id'] -ne $Outputs.cognito_client_id.value -or 'openid' -notin ($claims['scope'] -split ' ') -or -not $claims['sub'] -or [long]$claims['exp'] -le [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()) {
        throw 'Pipeline requires an unexpired Cognito access token with openid scope.'
    }
    if (-not (Get-Command aws -ErrorAction SilentlyContinue)) { throw 'Pipeline requires AWS CLI v2 and operator credentials.' }
    function Invoke-PipelineAWS([string[]]$Arguments, [hashtable]$InputObject = $null) {
        $temporary = $null
        try {
            if ($null -ne $InputObject) {
                $temporary = [IO.Path]::GetTempFileName()
                [IO.File]::WriteAllText($temporary, ($InputObject | ConvertTo-Json -Depth 12 -Compress), [Text.UTF8Encoding]::new($false))
                $Arguments += @('--cli-input-json', "file://$temporary")
            }
            $raw = & aws @Arguments --region $region --output json --no-cli-pager --cli-connect-timeout 5 --cli-read-timeout 15 2>$null | Out-String
            if ($LASTEXITCODE -ne 0) { throw "AWS Pipeline command failed: $($Arguments[0]) $($Arguments[1])" }
            if ([string]::IsNullOrWhiteSpace($raw)) { return @{} }
            return $raw | ConvertFrom-Json -AsHashtable
        } finally { if ($temporary) { Remove-Item -LiteralPath $temporary -Force } }
    }
    function Read-PipelineNumber([string]$PK, [string]$SK, [string]$Attribute) {
        $item = Invoke-PipelineAWS @('dynamodb','get-item') @{ TableName = $table; ConsistentRead = $true; Key = @{ PK = @{S=$PK}; SK = @{S=$SK} } }
        if (-not $item.ContainsKey('Item') -or -not $item.Item.ContainsKey($Attribute)) { return 0 }
        return [long]$item.Item[$Attribute].N
    }
    $identity = Invoke-PipelineAWS @('sts','get-caller-identity')
    if ($identity.Account -ne $account) { throw 'Operator credentials do not match the Terraform account.' }
    $mapping = Invoke-PipelineAWS @('lambda','get-event-source-mapping','--uuid',[string]$Outputs.click_event_source_mapping_id.value)
    if ($mapping.EventSourceArn -ne $Outputs.click_queue_arn.value -or $mapping.FunctionArn -ne $alias -or $mapping.State -ne 'Enabled' -or $mapping.BatchSize -ne 10 -or 'ReportBatchItemFailures' -notin $mapping.FunctionResponseTypes) { throw 'Unexpected ESM configuration.' }
    $attributes = (Invoke-PipelineAWS @('sqs','get-queue-attributes','--queue-url',$queueURL,'--attribute-names','VisibilityTimeout','RedrivePolicy')).Attributes
    $redrive = $attributes.RedrivePolicy | ConvertFrom-Json -AsHashtable
    if ([int]$attributes.VisibilityTimeout -ne 30 -or [int]$redrive.maxReceiveCount -ne 3 -or $redrive.deadLetterTargetArn -ne $Outputs.click_dlq_arn.value) { throw 'Unexpected source queue configuration.' }
    $dlqAttributes = (Invoke-PipelineAWS @('sqs','get-queue-attributes','--queue-url',$dlqURL,'--attribute-names','MessageRetentionPeriod','ApproximateNumberOfMessages','ApproximateNumberOfMessagesNotVisible')).Attributes
    if ([int]$dlqAttributes.MessageRetentionPeriod -ne 1209600 -or [int]$dlqAttributes.ApproximateNumberOfMessages -ne 0 -or [int]$dlqAttributes.ApproximateNumberOfMessagesNotVisible -ne 0) { throw 'Use a quiet test environment with an empty ClickDLQ before this scenario.' }
    $rule = Invoke-PipelineAWS @('events','describe-rule','--event-bus-name',$bus,'--name',"$prefix-link-clicked")
    $pattern = $rule.EventPattern | ConvertFrom-Json -AsHashtable
    $targets = Invoke-PipelineAWS @('events','list-targets-by-rule','--event-bus-name',$bus,'--rule',"$prefix-link-clicked")
    if ($rule.Arn -ne $Outputs.click_rule_arn.value -or $rule.State -ne 'ENABLED' -or @($pattern.source).Count -ne 1 -or $pattern.source[0] -ne 'urlshortener' -or @($pattern['detail-type']).Count -ne 1 -or $pattern['detail-type'][0] -ne 'link.clicked' -or @($targets.Targets | Where-Object { $_.Arn -eq $Outputs.click_queue_arn.value -and -not $_.ContainsKey('Input') -and -not $_.ContainsKey('InputPath') -and -not $_.ContainsKey('InputTransformer') }).Count -ne 1) { throw 'Unexpected EventBridge routing configuration.' }
    $transport = [Net.Http.HttpClientHandler]::new()
    $transport.AllowAutoRedirect = $false
    $client = [Net.Http.HttpClient]::new($transport)
    $client.Timeout = [TimeSpan]::FromSeconds(10)
    function Invoke-PipelineApi([string]$Method, [string]$Path, [string]$Body = '') {
        $request = [Net.Http.HttpRequestMessage]::new([Net.Http.HttpMethod]::new($Method), "$($base.AbsoluteUri.TrimEnd('/'))$Path")
        try {
            if ($Method -ne 'GET') { $request.Headers.Authorization = [Net.Http.Headers.AuthenticationHeaderValue]::new('Bearer', $token) }
            if ($Body) { $request.Content = [Net.Http.StringContent]::new($Body, [Text.Encoding]::UTF8, 'application/json') }
            $response = $client.SendAsync($request).GetAwaiter().GetResult()
            try { return @{Status = [int]$response.StatusCode; Body = $response.Content.ReadAsStringAsync().GetAwaiter().GetResult(); Location = $response.Headers.Location} }
            finally { $response.Dispose() }
        } finally { $request.Dispose() }
    }
    $code = $null
    try {
        $target = "https://example.org/pipeline/$([guid]::NewGuid().ToString('N'))"
        $created = Invoke-PipelineApi POST /links (@{url=$target} | ConvertTo-Json -Compress)
        if ($created.Status -ne 201) { throw 'Pipeline create failed.' }
        $link = $created.Body | ConvertFrom-Json -AsHashtable
        if ($link.code -notmatch '^[A-Za-z0-9]{7}$' -or $link.target -ne $target -or $link.shortUrl -ne "$($base.AbsoluteUri.TrimEnd('/'))/$($link.code)") { throw 'Unexpected created link; refusing cleanup of an unverified code.' }
        $code = $link.code
        $beforeDay = [DateTimeOffset]::UtcNow.ToString('yyyy-MM-dd')
        $resolved = Invoke-PipelineApi GET "/$code"
        if ($resolved.Status -ne 301 -or $resolved.Location.AbsoluteUri -ne $target -or $resolved.Body -ne '') { throw 'Pipeline redirect failed.' }
        $afterDay = [DateTimeOffset]::UtcNow.ToString('yyyy-MM-dd')
        $counted = $false
        for ($attempt=0; $attempt -lt 30; $attempt++) {
            $clicks = Read-PipelineNumber "LINK#$code" META clicks
            if ($clicks -gt 1) { throw 'One redirect produced more than one committed click.' }
            if ($clicks -eq 1) { $counted=$true; break }
            Start-Sleep -Seconds 1
        }
        if (-not $counted) { throw 'HTTP-published event did not commit within the bounded wait.' }
        $daily = 0
        foreach ($day in @($beforeDay,$afterDay) | Select-Object -Unique) { $daily += Read-PipelineNumber "LINK#$code" "STAT#$day" count }
        if ($daily -ne 1) { throw 'Daily counter does not match the committed redirect.' }
        $poison = Invoke-PipelineAWS @('events','put-events') @{ Entries = @(@{ EventBusName=$bus; Source='urlshortener'; DetailType='link.clicked'; Detail=(@{code=$code;at='invalid-pipeline-timestamp'} | ConvertTo-Json -Compress) }) }
        if ($poison.FailedEntryCount -ne 0 -or $poison.Entries.Count -ne 1 -or -not $poison.Entries[0].EventId) { throw 'Poison event was not accepted by EventBridge.' }
        $poisonID = $poison.Entries[0].EventId
        @{Check='poison published'; EventId=$poisonID; MaxReceiveCount=3} | ConvertTo-Json -Compress
        $found = $false
        for ($attempt=0; $attempt -lt 40; $attempt++) {
            $received = Invoke-PipelineAWS @('sqs','receive-message','--queue-url',$dlqURL,'--wait-time-seconds','5','--max-number-of-messages','10','--visibility-timeout','30','--message-system-attribute-names','ApproximateReceiveCount')
            if (-not $received.ContainsKey('Messages')) { continue }
            foreach ($message in $received.Messages) {
                $body = Read-PipelineEnvelope -Body $message.Body -EventID $poisonID
                if ($null -ne $body) {
                    if ($body.source -ne 'urlshortener' -or $body['detail-type'] -ne 'link.clicked' -or $body.detail.code -ne $code -or $body.detail.at -ne 'invalid-pipeline-timestamp') { throw 'The EventBridge envelope changed in transit.' }
                    @{Check='poison reached DLQ'; EventId=$poisonID; MessageId=$message.MessageId; ObservedReceiveCount=$message.Attributes.ApproximateReceiveCount} | ConvertTo-Json -Compress
                    $null = Invoke-PipelineAWS @('sqs','delete-message','--queue-url',$dlqURL,'--receipt-handle',$message.ReceiptHandle)
                    $found=$true
                } else {
                    # Never delete or leave another test/operator's message hidden.
                    $null = Invoke-PipelineAWS @('sqs','change-message-visibility','--queue-url',$dlqURL,'--receipt-handle',$message.ReceiptHandle,'--visibility-timeout','0')
                }
            }
            if ($found) { break }
            Start-Sleep -Seconds 5
        }
        if (-not $found) { throw "Poison event did not reach DLQ within the bounded wait; inspect event $poisonID manually." }
        if ((Read-PipelineNumber "LINK#$code" META clicks) -ne 1) { throw 'Poison event affected the committed count.' }
        $marker = Invoke-PipelineAWS @('dynamodb','get-item') @{ TableName=$table; ConsistentRead=$true; Key=@{ PK=@{S="CLICK#$poisonID"}; SK=@{S='CLICK'} } }
        if ($marker.ContainsKey('Item') -and $marker.Item.Count -gt 0) { throw 'Invalid event created an idempotency marker.' }
        Write-Host 'Pipeline E2E PASS: HTTP -> EventBridge -> SQS -> ESM -> DynamoDB; poison delivery reached DLQ with maxReceiveCount=3 configured.'
    } finally {
        try {
            if ($code) {
                $deleted = Invoke-PipelineApi DELETE "/links/$code"
                if ($deleted.Status -ne 200) { throw 'Pipeline cleanup failed.' }
                $null = Wait-DeletedLink -ReadLink { Invoke-PipelineApi GET "/$code" }
            }
        } finally { $client.Dispose(); $transport.Dispose() }
    }
}
