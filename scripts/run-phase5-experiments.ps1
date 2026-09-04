$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Push-Location $root
try {
    Write-Host "Running Phase 5 convergence and gossip-cost experiments..."
    go test ./internal/node -run '^TestPhase5' -v
    if ($LASTEXITCODE -ne 0) {
        throw "Phase 5 experiments failed"
    }

    Write-Host "Running the complete race-enabled suite..."
    go test -race ./...
    if ($LASTEXITCODE -ne 0) {
        throw "Race-enabled test suite failed"
    }
}
finally {
    Pop-Location
}
