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

$rootDir = Resolve-Path (Join-Path $PSScriptRoot "..\..")

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
