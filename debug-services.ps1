$services = @("catalog", "cart", "ordering", "inventory", "profiles", "reviews", "wishlists", "coupons")

Write-Host "Checking infrastructure..." -ForegroundColor Green
# Start infrastructure (if it's not running)
cd .. ; make up

Write-Host "Starting 8 microservices..." -ForegroundColor Green
foreach ($svc in $services) {
    # Open this specific microservice in a new PowerShell window without closing the terminal
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "`$host.UI.RawUI.WindowTitle='$svc Service'; cd ../src/services/$svc; go run ./cmd/server/"
}
Write-Host "All service terminal windows opened successfully! Wait for 'listening' logs on the black screens." -ForegroundColor Yellow
