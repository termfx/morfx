$ErrorActionPreference = "Stop"

function Invoke-CapturedNativeCommand {
	param(
		[Parameter(Mandatory = $true)]
		[string]$ExePath,
		[string[]]$ArgumentList = @(),
		[string]$Description = $ExePath
	)

	$output = & $ExePath @ArgumentList
	if ($LASTEXITCODE -ne 0) {
		throw "$Description failed with exit code $LASTEXITCODE"
	}
	return $output
}

function Write-Utf8NoBom {
	param(
		[Parameter(Mandatory = $true)]
		[string]$Path,
		[Parameter(Mandatory = $true)]
		[string]$Content
	)

	$encoding = New-Object System.Text.UTF8Encoding($false)
	[System.IO.File]::WriteAllText($Path, $Content, $encoding)
}

function Invoke-JsonTool {
	param(
		[Parameter(Mandatory = $true)]
		[string]$ExePath,
		[Parameter(Mandatory = $true)]
		[string]$InputPath,
		[Parameter(Mandatory = $true)]
		[string]$OutputPath
	)

	$scriptPath = Join-Path ([System.IO.Path]::GetTempPath()) ("morfx-stdin-" + [guid]::NewGuid().ToString("N") + ".cmd")
	$errorPath = "$OutputPath.err"
	try {
		$command = '@"' + $ExePath + '" < "' + $InputPath + '" > "' + $OutputPath + '" 2> "' + $errorPath + '"'
		Write-Utf8NoBom -Path $scriptPath -Content $command
		$proc = Start-Process -FilePath "cmd.exe" -ArgumentList "/d", "/c", $scriptPath -Wait -PassThru -NoNewWindow
		if ($proc.ExitCode -ne 0) {
			$errorText = ""
			if (Test-Path $errorPath) {
				$errorText = Get-Content $errorPath -Raw
			}
			if (-not $errorText -and (Test-Path $OutputPath)) {
				$errorText = Get-Content $OutputPath -Raw
			}
			throw "native command failed: $ExePath`n$errorText"
		}
	} finally {
		Remove-Item -LiteralPath $scriptPath -Force -ErrorAction SilentlyContinue
		Remove-Item -LiteralPath $errorPath -Force -ErrorAction SilentlyContinue
	}
}

$rootDir = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$artifactDir = if ($env:MORFX_ARTIFACT_DIR) { $env:MORFX_ARTIFACT_DIR } else { Join-Path $rootDir "artifacts\dogfood" }
$binDir = if ($env:MORFX_BIN_DIR) { $env:MORFX_BIN_DIR } else { Join-Path $rootDir "bin" }

New-Item -ItemType Directory -Force -Path $artifactDir | Out-Null
Get-ChildItem -Path $artifactDir -File -ErrorAction SilentlyContinue | Where-Object { $_.Extension -in ".go", ".json", ".txt" } | Remove-Item -Force

$requiredBins = @("morfx", "query", "replace", "file_query", "apply", "recipe")
foreach ($bin in $requiredBins) {
	$path = Join-Path $binDir "$bin.exe"
	if (-not (Test-Path $path)) {
		throw "missing binary: $path"
	}
}

Write-Utf8NoBom -Path (Join-Path $artifactDir "morfx-help.txt") -Content ((Invoke-CapturedNativeCommand -ExePath (Join-Path $binDir "morfx.exe") -ArgumentList @("--help") -Description "morfx --help") -join [Environment]::NewLine)
Write-Utf8NoBom -Path (Join-Path $artifactDir "apply-help.txt") -Content ((Invoke-CapturedNativeCommand -ExePath (Join-Path $binDir "apply.exe") -ArgumentList @("--help") -Description "apply --help") -join [Environment]::NewLine)
Write-Utf8NoBom -Path (Join-Path $artifactDir "recipe-help.txt") -Content ((Invoke-CapturedNativeCommand -ExePath (Join-Path $binDir "recipe.exe") -ArgumentList @("--help") -Description "recipe --help") -join [Environment]::NewLine)

$tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("morfx-standalone-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tmpDir | Out-Null

try {
	$sampleFile = Join-Path $tmpDir "sample.go"
	Write-Utf8NoBom -Path $sampleFile -Content @"
package sample

func HelloUser() string {
	return "hello"
}
"@

	$queryPayload = @{
		language = "go"
		path = $sampleFile
		query = @{
			type = "function"
			name = "Hello*"
		}
	} | ConvertTo-Json -Compress
	$queryJson = Join-Path $tmpDir "query.json"
	Write-Utf8NoBom -Path $queryJson -Content $queryPayload
	Invoke-JsonTool -ExePath (Join-Path $binDir "query.exe") -InputPath $queryJson -OutputPath (Join-Path $artifactDir "query.json")
	$queryOutput = Get-Content (Join-Path $artifactDir "query.json") -Raw
	if ($queryOutput -notmatch '"matches"' -or $queryOutput -notmatch 'HelloUser') {
		throw "query smoke check failed"
	}

	$replacePayload = @{
		language = "go"
		path = $sampleFile
		target = @{
			type = "function"
			name = "HelloUser"
		}
		replacement = 'func HelloUser() string { return "updated" }'
	} | ConvertTo-Json -Compress
	$replaceJson = Join-Path $tmpDir "replace.json"
	Write-Utf8NoBom -Path $replaceJson -Content $replacePayload
	Invoke-JsonTool -ExePath (Join-Path $binDir "replace.exe") -InputPath $replaceJson -OutputPath (Join-Path $artifactDir "replace.json")
	$sampleContent = Get-Content $sampleFile -Raw
	if ($sampleContent -notmatch 'updated') {
		throw "replace smoke check failed"
	}

	$fileQueryPayload = @{
		scope = @{
			path = $tmpDir
			include = @("**/*.go")
			language = "go"
			max_files = 10
		}
		query = @{
			type = "function"
			name = "Hello*"
		}
	} | ConvertTo-Json -Compress
	$fileQueryJson = Join-Path $tmpDir "file_query.json"
	Write-Utf8NoBom -Path $fileQueryJson -Content $fileQueryPayload
	Invoke-JsonTool -ExePath (Join-Path $binDir "file_query.exe") -InputPath $fileQueryJson -OutputPath (Join-Path $artifactDir "file_query.json")
	$fileQueryOutput = Get-Content (Join-Path $artifactDir "file_query.json") -Raw
	if ($fileQueryOutput -notmatch '"files"' -or $fileQueryOutput -notmatch 'HelloUser') {
		throw "file_query smoke check failed"
	}

	$recipePayload = @{
		name = "replace-hello-recipe"
		dry_run = $true
		min_confidence = 0.85
		steps = @(
			@{
				name = "replace hello function"
				method = "replace"
				scope = @{
					path = $tmpDir
					include = @("**/*.go")
					language = "go"
					max_files = 10
				}
				target = @{
					type = "function"
					name = "HelloUser"
				}
				replacement = 'func HelloUser() string { return "recipe" }'
			}
		)
	} | ConvertTo-Json -Compress -Depth 10
	$recipeJson = Join-Path $tmpDir "recipe.json"
	Write-Utf8NoBom -Path $recipeJson -Content $recipePayload
	Invoke-JsonTool -ExePath (Join-Path $binDir "recipe.exe") -InputPath $recipeJson -OutputPath (Join-Path $artifactDir "recipe.json")
	$recipeOutput = Get-Content (Join-Path $artifactDir "recipe.json") -Raw
	if ($recipeOutput -notmatch '"steps_run"' -or $recipeOutput -notmatch 'replace hello function') {
		throw "recipe smoke check failed"
	}
	$sampleAfterRecipe = Get-Content $sampleFile -Raw
	if ($sampleAfterRecipe -match 'recipe') {
		throw "recipe dry-run mutated the sample file"
	}

	Copy-Item -Path $sampleFile -Destination (Join-Path $artifactDir "sample.after.txt") -Force
	Write-Host "Standalone smoke completed. Artifacts written to $artifactDir"
} finally {
	Remove-Item -LiteralPath $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
