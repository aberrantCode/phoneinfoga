#Requires -Version 7.0
<#
.SYNOPSIS
    Local pre-PR validation gate for phoneinfoga.
.DESCRIPTION
    Runs the repo's Go test suite in short mode (`go test -short ./...`) and
    fails the push if any test fails. Invoked by scripts/git-hooks/pre-push on
    every push that carries commits. Short mode skips long/integration tests for
    a fast, deterministic gate. If the Go toolchain is not on PATH the gate
    reports and exits 0 (present-and-wired).
.PARAMETER Json
    Reserved for parity with the shared gate surface; unused here.
.NOTES
    Exit codes: 0 = tests passed (or go unavailable), 1 = tests failed,
    2 = execution error.
#>
[CmdletBinding()]
param([switch]$Json)
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot

$goCmd = Get-Command go -ErrorAction SilentlyContinue
if (-not $goCmd) {
    Write-Host 'validate: go toolchain not on PATH -- skipping test gate (present-and-wired).'
    exit 0
}

Push-Location $repoRoot
try {
    & $goCmd.Source test -short ./...
    $code = $LASTEXITCODE
}
finally {
    Pop-Location
}

if ($code -ne 0) {
    Write-Host "validate: go test FAILED (exit $code)."
    exit 1
}
Write-Host 'validate: go test -short passed.'
exit 0
