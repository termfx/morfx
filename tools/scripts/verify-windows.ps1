$ErrorActionPreference = "Stop"

function Invoke-NativeCommand {
	param(
		[Parameter(Mandatory = $true)]
		[string]$FilePath,
		[string[]]$ArgumentList = @(),
		[string]$Description = $FilePath
	)

	& $FilePath @ArgumentList
	if ($LASTEXITCODE -ne 0) {
		throw "$Description failed with exit code $LASTEXITCODE"
	}
}

function Get-CompilerExecutable {
	param(
		[Parameter(Mandatory = $true)]
		[string]$CompilerCommand
	)

	$trimmed = $CompilerCommand.Trim()
	if (-not $trimmed) {
		return $null
	}

	$parts = $trimmed -split '\s+', 2
	return $parts[0]
}

$rootDir = Resolve-Path (Join-Path $PSScriptRoot "..\..")

if (-not $env:CGO_ENABLED) {
	$env:CGO_ENABLED = "1"
}
if ($env:CGO_ENABLED -ne "1") {
	throw "verify-windows.ps1 requires CGO_ENABLED=1. Unset CGO_ENABLED or set it to 1 before running."
}

if (-not $env:CC) {
	$zig = Get-Command zig -ErrorAction SilentlyContinue
	if ($zig) {
		$env:CC = "zig cc -target x86_64-windows-gnu"
		Write-Host "Using Zig for CC: $env:CC"
	} else {
		throw "verify-windows.ps1 requires a Windows CGO compiler. Install Zig and ensure `zig` is on PATH, or set CC explicitly (for example: `$env:CC = 'zig cc -target x86_64-windows-gnu'`)."
	}
}

$compilerExe = Get-CompilerExecutable -CompilerCommand $env:CC
if (-not $compilerExe -or -not (Get-Command $compilerExe -ErrorAction SilentlyContinue)) {
	throw "verify-windows.ps1 could not resolve the compiler from CC='$env:CC'. Install the compiler or set CC to a valid command before running."
}

Push-Location $rootDir
try {
	Write-Host "Downloading and verifying dependencies..."
	Invoke-NativeCommand -FilePath "go" -ArgumentList @("mod", "download") -Description "go mod download"
	Invoke-NativeCommand -FilePath "go" -ArgumentList @("mod", "verify") -Description "go mod verify"

	Write-Host "Running tests..."
	Invoke-NativeCommand -FilePath "go" -ArgumentList @("test", "./...") -Description "go test ./..."

	Write-Host "Running go vet..."
	Invoke-NativeCommand -FilePath "go" -ArgumentList @("vet", "./...") -Description "go vet ./..."

	& (Join-Path $rootDir "tools\scripts\build-standalone.ps1")
	& (Join-Path $rootDir "tools\scripts\smoke-standalone.ps1")

	Write-Host "Windows verification complete."
} finally {
	Pop-Location
}
