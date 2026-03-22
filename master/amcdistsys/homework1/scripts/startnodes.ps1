# Usage: .\scripts\startnodes.ps1 <config_file> <first_index> <last_index>
# Example: .\scripts\startnodes.ps1 config.txt 0 2

param(
    [Parameter(Mandatory)][string]$Config,
    [Parameter(Mandatory)][int]$First,
    [Parameter(Mandatory)][int]$Last
)

$Root = Split-Path -Parent $PSScriptRoot
$Binary = Join-Path $Root "bcastnode.exe"

Write-Host "Building bcastnode..."
Push-Location $Root
go build -o bcastnode.exe ./cmd/bcastnode
if ($LASTEXITCODE -ne 0) { Write-Error "Build failed"; exit 1 }
Pop-Location
Write-Host "Build complete."

$jobs = @()
for ($idx = $First; $idx -le $Last; $idx++) {
    $job = Start-Process -FilePath $Binary -ArgumentList $Config, $idx -PassThru -NoNewWindow
    $jobs += $job
    Write-Host "Started node $idx (pid $($job.Id))"
}

Write-Host "All nodes launched. Waiting for completion..."
$jobs | ForEach-Object { $_.WaitForExit() }
Write-Host "All nodes done."
