param(
    [string]$Jwt = ""
)

$BaseUrl = "http://localhost:8080/api"
$DefaultJwt = "PASTE_JWT_HERE"
$EffectiveJwt = if ($Jwt -ne "") { $Jwt } else { $DefaultJwt }

$Headers = @{
    "Authorization" = "Bearer $EffectiveJwt"
    "Content-Type"  = "application/json"
}

function Test-Endpoint {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Path,
        $Body = $null,
        [switch]$AllowFailure
    )

    Write-Host "Testing $Name... " -NoNewline
    try {
        $Params = @{
            Uri         = "$BaseUrl$Path"
            Method      = $Method
            Headers     = $Headers
            ErrorAction = "Stop"
        }
        if ($null -ne $Body) { $Params.Body = ($Body | ConvertTo-Json) }

        $Response = Invoke-RestMethod @Params
        Write-Host "OK" -ForegroundColor Green
        return $Response
    } catch {
        if ($AllowFailure) {
            Write-Host "SKIP" -ForegroundColor Yellow
            Write-Host "  -> $($_.Exception.Message)" -ForegroundColor DarkYellow
            return $null
        }
        Write-Host "FAIL" -ForegroundColor Red
        Write-Error $_.Exception.Message
        exit 1
    }
}

if ($EffectiveJwt -eq "PASTE_JWT_HERE") {
    $seed = Get-Random
    $regBody = @{
        username = "testuser_$seed"
        email    = "test_$seed@example.com"
        password = "password123"
    }

    Write-Host "Testing Register... " -NoNewline
    $reg = Test-Endpoint "Register" "POST" "/auth/register" $regBody
    if ($null -eq $reg -or [string]::IsNullOrWhiteSpace($reg.token)) {
        Write-Host "FAIL" -ForegroundColor Red
        Write-Error "Register did not return token"
        exit 1
    }
    Write-Host "OK" -ForegroundColor Green

    Write-Host "Testing Login... " -NoNewline
    $loginBody = @{
        email    = $regBody.email
        password = $regBody.password
    }
    $login = Test-Endpoint "Login" "POST" "/auth/login" $loginBody
    if ($null -eq $login -or [string]::IsNullOrWhiteSpace($login.token)) {
        Write-Host "FAIL" -ForegroundColor Red
        Write-Error "Login did not return token"
        exit 1
    }
    Write-Host "OK" -ForegroundColor Green

    $EffectiveJwt = $login.token
    $Headers["Authorization"] = "Bearer $EffectiveJwt"
    Write-Host "JWT acquired via /auth/register + /auth/login" -ForegroundColor Cyan
}

# --- ВЫПОЛНЕНИЕ ТЕСТОВ ---

# 1. Проверка Health
Test-Endpoint "Health Check" "GET" "/health" -AllowFailure

# 2. Проверка профиля
$Me = Test-Endpoint "Get Me" "GET" "/me"
Write-Host "Logged in as: $($Me.username)"

# 3. Создание гильдии
$GuildData = @{ name = "Automation Guild $(Get-Random)" }
$NewGuild = Test-Endpoint "Create Guild" "POST" "/guilds" $GuildData
$GID = $NewGuild.id
Write-Host "Created Guild ID: $GID"

# 4. Создание канала
$ChannelData = @{ name = "general"; type = "text" }
$NewChannel = Test-Endpoint "Create Channel" "POST" "/guilds/$GID/channels" $ChannelData
$CID = $NewChannel.id
Write-Host "Created Channel ID: $CID"

# 5. Список каналов
$Channels = Test-Endpoint "List Channels" "GET" "/guilds/$GID/channels"
if ($null -eq $Channels.channels -or $Channels.channels.Count -lt 1) {
    Write-Host "FAIL" -ForegroundColor Red
    Write-Error "Guild channels list is empty"
    exit 1
}

# 6. Отправка сообщения в канал
$MessageData = @{ content = "hello from smoke test $(Get-Random)" }
$Message = Test-Endpoint "Create Message" "POST" "/channels/$CID/messages" $MessageData
if ([string]::IsNullOrWhiteSpace($Message.content)) {
    Write-Host "FAIL" -ForegroundColor Red
    Write-Error "Message content is empty in response"
    exit 1
}

# 7. Получение последних сообщений
$RecentMessages = Test-Endpoint "Get Recent Messages" "GET" "/channels/$CID/messages"
if ($null -eq $RecentMessages.messages -or $RecentMessages.messages.Count -lt 1) {
    Write-Host "FAIL" -ForegroundColor Red
    Write-Error "Recent messages are empty"
    exit 1
}

# 8. Проверка Voice Token
$VoiceData = @{ guild_id = [int]$GID; room_name = "General" }
$VoiceResponse = Test-Endpoint "Join Voice" "POST" "/voice/join-token" $VoiceData
if ($VoiceResponse.token) { Write-Host "Voice Token received" -ForegroundColor Cyan }

# 9. Негативный тест rate limit (ожидаем 429 на части запросов)
Write-Host "Testing Message Rate Limit... " -NoNewline
$rateLimitHit = $false
for ($i = 0; $i -lt 15; $i++) {
    $payload = @{ content = "rate-limit-$i-$(Get-Random)" }
    try {
        Invoke-WebRequest -Uri "$BaseUrl/channels/$CID/messages" -Method "POST" -Headers $Headers -Body ($payload | ConvertTo-Json) -ErrorAction Stop | Out-Null
    } catch {
        $statusCode = 0
        if ($null -ne $_.Exception.Response) {
            $statusCode = [int]$_.Exception.Response.StatusCode
        }
        if ($statusCode -eq 429) {
            $rateLimitHit = $true
            break
        }
    }
}
if (-not $rateLimitHit) {
    Write-Host "FAIL" -ForegroundColor Red
    Write-Error "Rate limit was not triggered after burst message send"
    exit 1
}
Write-Host "OK" -ForegroundColor Green

Write-Host "`n--- SMOKE TEST PASSED ---" -ForegroundColor Green