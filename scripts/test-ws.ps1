param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$WsUrl = "ws://localhost:8080/ws",
    [int]$TimeoutSeconds = 8
)

$ErrorActionPreference = "Stop"

function ConvertTo-JsonCompact {
    param($Value)
    return ($Value | ConvertTo-Json -Depth 10 -Compress)
}

function Connect-WebSocket {
    param(
        [string]$Url,
        [string]$Jwt
    )

    $uri = [System.Uri]::new($Url)
    $ws = [System.Net.WebSockets.ClientWebSocket]::new()
    $ws.Options.SetRequestHeader("Authorization", "Bearer $Jwt")
    [void]$ws.ConnectAsync($uri, [System.Threading.CancellationToken]::None).GetAwaiter().GetResult()
    return $ws
}

function Send-WSJson {
    param(
        [System.Net.WebSockets.ClientWebSocket]$Ws,
        [object]$Payload
    )
    $json = ConvertTo-JsonCompact $Payload
    $bytes = [System.Text.Encoding]::UTF8.GetBytes($json)
    $segment = [System.ArraySegment[byte]]::new($bytes)
    [void]$Ws.SendAsync(
        $segment,
        [System.Net.WebSockets.WebSocketMessageType]::Text,
        $true,
        [System.Threading.CancellationToken]::None
    ).GetAwaiter().GetResult()
}

function Receive-WSJson {
    param(
        [System.Net.WebSockets.ClientWebSocket]$Ws,
        [int]$TimeoutSeconds
    )

    $buffer = New-Object byte[] 8192
    $segment = [System.ArraySegment[byte]]::new($buffer)
    $cts = [System.Threading.CancellationTokenSource]::new()
    $cts.CancelAfter([TimeSpan]::FromSeconds($TimeoutSeconds))

    try {
        $result = $Ws.ReceiveAsync($segment, $cts.Token).GetAwaiter().GetResult()
    } catch {
        if ($cts.IsCancellationRequested) {
            return $null
        }
        return @{ event = "SOCKET_ERROR"; payload = @{ message = $_.Exception.Message } }
    } finally {
        $cts.Dispose()
    }

    if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
        return @{ event = "SOCKET_CLOSED"; payload = @{} }
    }

    $jsonText = [System.Text.Encoding]::UTF8.GetString($buffer, 0, $result.Count)
    try {
        return ($jsonText | ConvertFrom-Json)
    } catch {
        return @{ event = "NON_JSON"; payload = @{ raw = $jsonText } }
    }
}

function Wait-ForEvent {
    param(
        [System.Net.WebSockets.ClientWebSocket]$Ws,
        [string]$EventName,
        [int]$TimeoutSeconds
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $msg = Receive-WSJson -Ws $Ws -TimeoutSeconds 1
        if ($null -eq $msg) { continue }
        if ($msg.event -eq $EventName) { return $msg }
        if ($msg.Event -eq $EventName) { return $msg }
    }
    return $null
}

Write-Host "=== WS E2E: register/login ===" -ForegroundColor Cyan
$seed = Get-Random
$registerBody = @{
    username = "ws_e2e_$seed"
    email    = "ws_e2e_$seed@example.com"
    password = "password123"
}
$reg = Invoke-RestMethod -Uri "$BaseUrl/api/auth/register" -Method Post -Body (ConvertTo-JsonCompact $registerBody) -ContentType "application/json"
$login = Invoke-RestMethod -Uri "$BaseUrl/api/auth/login" -Method Post -Body (ConvertTo-JsonCompact @{ email = $registerBody.email; password = $registerBody.password }) -ContentType "application/json"
$jwt = $login.token
if ([string]::IsNullOrWhiteSpace($jwt)) { throw "JWT not returned from login" }

Write-Host "=== WS E2E: guild/channel setup ===" -ForegroundColor Cyan
$headers = @{ Authorization = "Bearer $jwt" }
$guild = Invoke-RestMethod -Uri "$BaseUrl/api/guilds" -Method Post -Headers $headers -Body (ConvertTo-JsonCompact @{ name = "WS E2E Guild $seed" }) -ContentType "application/json"
$guildId = [int]$guild.id
$channel = Invoke-RestMethod -Uri "$BaseUrl/api/guilds/$guildId/channels" -Method Post -Headers $headers -Body (ConvertTo-JsonCompact @{ name = "general"; type = "text" }) -ContentType "application/json"
$channelId = [int]$channel.id
Write-Host "Guild: $guildId, Channel: $channelId" -ForegroundColor Green

$a = $null
$b = $null
$c = $null
try {
    Write-Host "=== WS E2E: connect clients A/B/C ===" -ForegroundColor Cyan
    $a = Connect-WebSocket -Url $WsUrl -Jwt $jwt
    $b = Connect-WebSocket -Url $WsUrl -Jwt $jwt
    $c = Connect-WebSocket -Url $WsUrl -Jwt $jwt

    Send-WSJson -Ws $a -Payload @{ op = 10; event = "SUBSCRIBE_CHANNEL"; payload = @{ channel_id = $channelId } }
    Send-WSJson -Ws $b -Payload @{ op = 10; event = "SUBSCRIBE_CHANNEL"; payload = @{ channel_id = $channelId } }

    $ackA = Wait-ForEvent -Ws $a -EventName "CHANNEL_SUBSCRIBED" -TimeoutSeconds $TimeoutSeconds
    $ackB = Wait-ForEvent -Ws $b -EventName "CHANNEL_SUBSCRIBED" -TimeoutSeconds $TimeoutSeconds
    if ($null -eq $ackA -or $null -eq $ackB) { throw "Did not receive CHANNEL_SUBSCRIBED ack for A or B" }

    Write-Host "=== WS E2E: send MESSAGE_CREATE from A ===" -ForegroundColor Cyan
    $content = "ws-e2e-message-$seed"
    Send-WSJson -Ws $a -Payload @{
        op = 1
        event = "MESSAGE_CREATE"
        payload = @{
            channel_id = $channelId
            content    = $content
        }
    }

    $recvB = Wait-ForEvent -Ws $b -EventName "MESSAGE_CREATE" -TimeoutSeconds $TimeoutSeconds
    if ($null -eq $recvB) { throw "Client B did not receive MESSAGE_CREATE" }

    $recvC = Wait-ForEvent -Ws $c -EventName "MESSAGE_CREATE" -TimeoutSeconds 2
    if ($null -ne $recvC) { throw "Client C unexpectedly received MESSAGE_CREATE (broadcast leak)" }

    $payload = $recvB.payload
    if ($payload.content -ne $content) { throw "Client B received wrong content" }
    if ([int]$payload.channel_id -ne $channelId) { throw "Client B received wrong channel_id" }

    Write-Host "WS targeted delivery OK: B got message, C did not." -ForegroundColor Green
    Write-Host "--- WS E2E PASSED ---" -ForegroundColor Green
}
finally {
    foreach ($ws in @($a, $b, $c)) {
        if ($null -ne $ws) {
            try {
                if ($ws.State -eq [System.Net.WebSockets.WebSocketState]::Open) {
                    $ws.CloseAsync(
                        [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
                        "bye",
                        [System.Threading.CancellationToken]::None
                    ).GetAwaiter().GetResult()
                }
            } catch { }
            $ws.Dispose()
        }
    }
}
