# run-server.ps1 - Unified launcher for Indexium Financial AI (Frontend + Backend)
# Usage:  .\run-server.ps1 [-Port 3000]
#         .\run-server.ps1 [-Query "your query"]
#         .\run-server.ps1 (starts backend server & UI browser on :8080)

param(
    [string]$Query = "",
    [int]$Port = 8080
)

# Use $PSScriptRoot to avoid hardcoding or relying on Get-Location
$RootPath = $PSScriptRoot
$GeminiPath = Join-Path $RootPath "Gemini"

if (-not (Test-Path $GeminiPath)) {
    Write-Host "[Error] Gemini directory not found at $GeminiPath" -ForegroundColor Red
    exit 1
}

Set-Location $GeminiPath

Write-Host "====================================================" -ForegroundColor Cyan
Write-Host "  INDEXIUM FINANCIAL AI - FULL STACK WORKSPACE     " -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Cyan
Write-Host "Backend: Go (Gemini Agent)" -ForegroundColor Gray
Write-Host "Frontend: HTML/JS (Vanilla)" -ForegroundColor Gray
Write-Host "Working Directory: $GeminiPath" -ForegroundColor Gray

# 1. Check for Go
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Host "[Error] Go is not installed or not in PATH." -ForegroundColor Red
    Write-Host "Please install Go from https://go.dev/dl/ before running." -ForegroundColor Yellow
    exit 1
}

# 2. Check for .env
if (-not (Test-Path '.env')) {
    if (Test-Path '.env.example') {
        Write-Host "[Warning] No .env found. Copying from .env.example ..." -ForegroundColor Yellow
        Copy-Item '.env.example' '.env'
        Write-Host "[Info] Please EDIT the .env file and add your real API keys before continuing." -ForegroundColor Yellow
        notepad '.env'
        Write-Host "After saving .env, re-run this script." -ForegroundColor Yellow
        exit 0
    } else {
        Write-Host "[Error] .env or .env.example not found in $GeminiPath" -ForegroundColor Red
        exit 1
    }
}

# 3. Check for Redis
$RedisAddr = "localhost:6379"
$envContent = Get-Content '.env'
foreach ($line in $envContent) {
    if ($line -match '^REDIS_ADDR=(.+)$') {
        $RedisAddr = $matches[1].Trim()
    }
}

$tcpClient = New-Object System.Net.Sockets.TcpClient
$parts = $RedisAddr.Split(':')
$redisHost = $parts[0]
$redisPort = if ($parts.Length -gt 1) { [int]$parts[1] } else { 6379 }

try {
    $wait = $tcpClient.ConnectAsync($redisHost, $redisPort)
    if (-not $wait.Wait(300)) { throw "Timeout" }
    Write-Host "[Redis] Detected running on $RedisAddr" -ForegroundColor Gray
} catch {
    Write-Host "[Redis] Not detected. Multi-chat sessions will be in-memory only." -ForegroundColor Yellow
} finally {
    $tcpClient.Close()
}

$FrontendPath = Join-Path $RootPath "frontend"
if (Test-Path $FrontendPath) {
    Write-Host "[Build] Building Frontend (dist)..." -ForegroundColor Yellow
    Push-Location $FrontendPath
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[Error] Frontend build failed." -ForegroundColor Red
        Pop-Location
        exit $LASTEXITCODE
    }
    Pop-Location
    Write-Host "[Success] Frontend built." -ForegroundColor Green
}

Write-Host "[Build] Building/Verifying Backend Executable..." -ForegroundColor Yellow
go build -o server.exe cmd/gemini-cli/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "[Error] Compilation failed." -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[Success] Backend compiled successfully." -ForegroundColor Green

if ($Query) {
    Write-Host "Running one-shot query: $Query" -ForegroundColor Green
    .\server.exe $Query
} else {
    Write-Host ""
    $env:PORT = "$Port"

    Write-Host "[Start] Starting Backend Server and Serving Frontend..." -ForegroundColor Green
    Write-Host "[URL] Local URL: http://localhost:$Port" -ForegroundColor White
    Write-Host "[Tip] The browser will open automatically in 3 seconds..." -ForegroundColor Gray
    Write-Host "----------------------------------------------------" -ForegroundColor Gray

    # Start browser in background
    Start-Sleep -Seconds 3
    Start-Process "http://localhost:$Port"

    # Run Backend Server
    .\server.exe -server -port $Port
}
