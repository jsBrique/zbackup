# 在 PowerShell 中执行该脚本，即可编译 Linux/AMD64 版本
param(
    [string]$Output = "build/zbackup"
)

$ErrorActionPreference = "Stop"

if (!(Test-Path -Path "build")) {
    New-Item -ItemType Directory -Path "build" | Out-Null
}

$env:GOOS = "linux"
$env:GOARCH = "amd64"

Write-Host "开始编译 Linux/AMD64 版本 -> $Output"
go build -o $Output ./cmd/zbackup
Write-Host "编译完成：$Output"

