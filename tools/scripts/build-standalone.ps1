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
$binDir = Join-Path $rootDir "bin"
$tools = @("query", "replace", "delete", "insert_before", "insert_after", "append", "file_query", "file_replace", "file_delete", "apply", "recipe")

Push-Location $rootDir
try {
	New-Item -ItemType Directory -Force -Path $binDir | Out-Null

	Write-Host "Building morfx..."
	Invoke-NativeCommand -FilePath "go" -ArgumentList @("build", "-o", (Join-Path $binDir "morfx.exe"), "./cmd/morfx") -Description "go build morfx"

	foreach ($tool in $tools) {
		Write-Host "Building $tool..."
		Invoke-NativeCommand -FilePath "go" -ArgumentList @("build", "-o", (Join-Path $binDir "$tool.exe"), "./cmd/$tool") -Description "go build $tool"
	}

	Write-Host "Standalone tools built in $binDir"
} finally {
	Pop-Location
}
