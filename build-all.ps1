# 在 PowerShell 中运行本脚本，即可一次性交叉编译常见平台的 zbackup 可执行文件
$ErrorActionPreference = "Stop"

$targets = @(
    @{ OS = "linux";  Arch = "amd64"; Ext = ""       },
    @{ OS = "linux";  Arch = "arm64"; Ext = ""       },
    @{ OS = "darwin"; Arch = "amd64"; Ext = ""       },
    @{ OS = "darwin"; Arch = "arm64"; Ext = ""       },
    @{ OS = "windows"; Arch = "amd64"; Ext = ".exe"  },
    @{ OS = "windows"; Arch = "arm64"; Ext = ".exe"  }
)

if (!(Test-Path -Path "build")) {
    New-Item -ItemType Directory -Path "build" | Out-Null
}

foreach ($target in $targets) {
    $env:GOOS = $target.OS
    $env:GOARCH = $target.Arch
    $ext = $target.Ext
    $output = "build/zbackup-$($target.OS)-$($target.Arch)$ext"

    Write-Host "编译 $($target.OS)/$($target.Arch) -> $output"
    go build -o $output ./cmd/zbackup
}

Write-Host "编译完成，全部二进制位于 build/ 目录"

