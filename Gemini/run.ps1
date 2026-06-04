# run.ps1 - Helper to start the Indexium Financial AI Agent (Gemini)
# Usage:  Right-click -> Run with PowerShell, or .\run.ps1

param(
    [switch]$ServerOnly = $true,
    [string]$Query = ""
)

$ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectDir

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Indexium Financial AI Agent (Gemini)  " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Working dir: $ProjectDir" -ForegroundColor Gray

# Check for Go
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Error: Go is not installed or not in PATH." -ForegroundColor Red
    exit
}

# Check for .env
if (-not (Test-Path '.env')) {
    if (Test-Path '.env.example') {
        Write-Host "⚠️  No .env found. Copying from .env.example ..." -ForegroundColor Yellow
        Copy-Item '.env.example' '.env'
        Write-Host "📝 Please EDIT the .env file and add your real API keys before continuing." -ForegroundColor Yellow
        notepad '.env'
        Write-Host "After saving .env, re-run this script." -ForegroundColor Yellow
        exit
    }
}

# Check for Redis
$RedisAddr = "localhost:6379"
if (Test-Path '.env') {
    $envContent = Get-Content '.env'
    foreach ($line in $envContent) {
        if ($line -match '^REDIS_ADDR=(.+)$') { $RedisAddr = $matches[1].Trim() }
    }
}

$tcpClient = New-Object System.Net.Sockets.TcpClient
$parts = $RedisAddr.Split(':')
$redisHost = $parts[0]
$port = if ($parts.Length -gt 1) { [int]$parts[1] } else { 6379 }

try {
    $wait = $tcpClient.ConnectAsync($redisHost, $port)
    if (-not $wait.Wait(300)) { throw "Timeout" }
    Write-Host "✅ [Redis] Detected running on $RedisAddr" -ForegroundColor Gray
} catch {
    Write-Host "⚠️  [Redis] Not detected. Multi-chat sessions will be in-memory only." -ForegroundColor Yellow
} finally {
    $tcpClient.Close()
}

if ($Query) {
    Write-Host "Running one-shot query: $Query" -ForegroundColor Green
    go run cmd/gemini-cli/main.go $Query
} else {
    Write-Host ""
    Write-Host "🚀 Starting Backend Server & Serving Frontend..." -ForegroundColor Green
    Write-Host "🌐 Local URL: http://localhost:8080" -ForegroundColor White
    
    # Optional: Start browser
    Start-Sleep -Seconds 2
    Start-Process "http://localhost:8080"
    
    go run cmd/gemini-cli/main.go -server
}

