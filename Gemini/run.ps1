# run.ps1 - Helper to start the Indexium Financial AI Agent (Gemini)
# Usage:  Right-click -> Run with PowerShell, or .\run.ps1   or  .\run.ps1 -ServerOnly

param(
    [switch]$ServerOnly = $true,   # default to server (UI) mode
    [string]$Query = ""
)

$ProjectDir = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not (Test-Path (Join-Path $ProjectDir 'cmd\gemini-cli\main.go'))) {
    # If script is placed elsewhere, try known location
    $ProjectDir = 'C:\indexium\Term 2\Anthropics-Financial-Services\Gemini'
}

Set-Location $ProjectDir

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " Indexium Financial AI Agent (Gemini)  " -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Working dir: $ProjectDir" -ForegroundColor Gray

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
