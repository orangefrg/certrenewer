if ((Get-Item -Path ".").BaseName -ne "build") {
    Write-Error "This script must be run from the 'build' directory."
    exit 1
}

$rootDir = (Get-Item -Path "../").BaseName

$cmdDir = "../cmd"
$binDir = "../bin"

if (-not (Test-Path -Path $binDir)) {
    New-Item -ItemType Directory -Path $binDir | Out-Null
}

if (-not (Test-Path -Path $cmdDir)) {
    Write-Error "cmd/ directory does not exist. Please check the project structure."
    exit 1
}

Push-Location $cmdDir

try {
    # Build for Windows
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    Write-Host "Building for Windows..."
    go build -o "$binDir/$rootDir.exe"

    # Build for Linux
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    Write-Host "Building for Linux..."
    go build -o "$binDir/$rootDir"

    Write-Host "Build completed successfully. Binaries are in the 'bin/' directory."

} catch {
    Write-Error "An error occurred during the build process: $_"
} finally {
    Pop-Location

    if ($env:GOOS) {
        Remove-Item -Path env:GOOS
    }
    if ($env:GOARCH) {
        Remove-Item -Path env:GOARCH
    }
}