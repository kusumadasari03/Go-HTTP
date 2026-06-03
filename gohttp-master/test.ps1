# Simple test script for our HTTP server (PowerShell)

Write-Host "🧪 Testing GoHTTP Server" -ForegroundColor Cyan
Write-Host "========================" -ForegroundColor Cyan

# Test different endpoints
Write-Host "📍 Testing root endpoint..." -ForegroundColor Yellow
go run cmd/client/main.go /

Write-Host ""
Write-Host "📍 Testing /hello endpoint..." -ForegroundColor Yellow
go run cmd/client/main.go /hello

Write-Host ""
Write-Host "📍 Testing /info endpoint..." -ForegroundColor Yellow
go run cmd/client/main.go /info

Write-Host ""
Write-Host "📍 Testing /json endpoint..." -ForegroundColor Yellow
go run cmd/client/main.go /json

Write-Host ""
Write-Host "📍 Testing 404 endpoint..." -ForegroundColor Yellow
go run cmd/client/main.go /nonexistent

Write-Host ""
Write-Host "✅ Tests completed!" -ForegroundColor Green