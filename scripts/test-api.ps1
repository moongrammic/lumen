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

# 4. Проверка Voice Token
$VoiceData = @{ guild_id = [int]$GID; room_name = "General" }
$VoiceResponse = Test-Endpoint "Join Voice" "POST" "/voice/join-token" $VoiceData
if ($VoiceResponse.token) { Write-Host "Voice Token received" -ForegroundColor Cyan }

Write-Host "`n--- SMOKE TEST PASSED ---" -ForegroundColor Green