# run.ps1 - Helper to start the Indexium Financial AI Agent (Gemini)
# Usage:  Right-click -> Run with PowerShell, or .\run.ps1   or  .\run.ps1 -ServerOnly

param(
    [switch]$ServerOnly = $true,   # default to server (UI) mode
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
    Write-Host "Please install Go from https://go.dev/dl/ before running this script." -ForegroundColor Yellow
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
    } else {
        Write-Host "❌ Error: .env or .env.example not found in $ProjectDir" -ForegroundColor Red
        exit
    }
}

# Check for Redis (Optional but recommended)
$RedisAddr = "localhost:6379"
# Try to parse from .env if possible
if (Test-Path '.env') {
    $envContent = Get-Content '.env'
    foreach ($line in $envContent) {
        if ($line -match '^REDIS_ADDR=(.+)$') {
            $RedisAddr = $matches[1].Trim()
        }
    }
}

$tcpClient = New-Object System.Net.Sockets.TcpClient
$parts = $RedisAddr.Split(':')
$host = $parts[0]
$port = if ($parts.Length -gt 1) { [int]$parts[1] } else { 6379 }

try {
    $wait = $tcpClient.ConnectAsync($host, $port)
    if (-not $wait.Wait(500)) { throw "Timeout" }
    Write-Host "✅ [Redis] Detected running on $RedisAddr" -ForegroundColor Gray
} catch {
    Write-Host "⚠️  [Redis] Not detected on $RedisAddr. Multi-chat sessions will be in-memory only." -ForegroundColor Yellow
} finally {
    $tcpClient.Close()
}

if ($Query) {
    Write-Host "Running one-shot query: $Query" -ForegroundColor Green
    go run cmd/gemini-cli/main.go $Query
} else {
    Write-Host "Starting in SERVER mode (with Web UI) on http://localhost:8080 ..." -ForegroundColor Green
    Write-Host "Open your browser to: http://localhost:8080" -ForegroundColor White
    Write-Host "Use the gear icon (⚙️) in UI to set additional OpenRouter keys if needed." -ForegroundColor Gray
    Write-Host "Press Ctrl+C to stop the server." -ForegroundColor Gray
    Write-Host ""
    go run cmd/gemini-cli/main.go -server
}

