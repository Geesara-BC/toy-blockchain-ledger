$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$binary = Join-Path $root "bin\blockchain-node.exe"

Push-Location $root
try {
    New-Item -ItemType Directory -Force -Path (Join-Path $root "bin") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $root "data\node1") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $root "data\node2") | Out-Null
    New-Item -ItemType Directory -Force -Path (Join-Path $root "data\node3") | Out-Null

    go build -o $binary .\cmd\node

    $nodes = @(
        @{ Name = "blockchain-node-1"; Port = 8081; Peers = "http://localhost:8082"; DbPath = ".\data\node1\blockchain.json" },
        @{ Name = "blockchain-node-2"; Port = 8082; Peers = "http://localhost:8081,http://localhost:8083"; DbPath = ".\data\node2\blockchain.json" },
        @{ Name = "blockchain-node-3"; Port = 8083; Peers = "http://localhost:8082"; DbPath = ".\data\node3\blockchain.json" }
    )

    foreach ($node in $nodes) {
        $arguments = "--addr=:$($node.Port) --peers=$($node.Peers) --db-path=$($node.DbPath)"
        Start-Process -FilePath $binary -ArgumentList $arguments -WorkingDirectory $root -WindowStyle Normal
        Write-Host "Started $($node.Name) on port $($node.Port)"
    }
}
finally {
    Pop-Location
}

Write-Host "Local cluster started: http://localhost:8081, http://localhost:8082, http://localhost:8083"