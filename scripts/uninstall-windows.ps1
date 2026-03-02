#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Keldris Agent Uninstaller for Windows

.DESCRIPTION
    Stops the Windows Service, removes the binary, and optionally
    removes configuration and data files.

.PARAMETER Purge
    Also remove configuration and data files.

.EXAMPLE
    .\uninstall-windows.ps1

.EXAMPLE
    .\uninstall-windows.ps1 -Purge
#>

param(
    [Parameter(Mandatory = $false)]
    [switch]$Purge
)

$ServiceName = "KeldrisAgent"
$BinaryName = "keldris-agent.exe"
$InstallDir = "$env:ProgramFiles\Keldris"
$ConfigDir = "$env:ProgramData\Keldris"

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Main {
    if (-not (Test-Administrator)) {
        Write-Err "This script must be run as Administrator"
        Write-Host "Right-click PowerShell and select 'Run as Administrator'"
        exit 1
    }

    Write-Info "Uninstalling Keldris Agent..."

    # Stop and remove Windows Service
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq "Running") {
            Write-Info "Stopping service..."
            Stop-Service -Name $ServiceName -Force
            Start-Sleep -Seconds 2
        }
        Write-Info "Removing service..."
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 1
    }

    # Remove binary
    $binaryPath = Join-Path $InstallDir $BinaryName
    if (Test-Path $binaryPath) {
        Write-Info "Removing binary..."
        Remove-Item -Path $binaryPath -Force
    }

    # Remove from PATH
    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($currentPath -like "*$InstallDir*") {
        $newPath = ($currentPath.Split(';') | Where-Object { $_ -ne $InstallDir }) -join ';'
        [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
        Write-Info "Removed from system PATH"
    }

    # Remove KELDRIS_CONFIG_DIR env var
    $configDirEnv = [Environment]::GetEnvironmentVariable("KELDRIS_CONFIG_DIR", "Machine")
    if ($configDirEnv) {
        [Environment]::SetEnvironmentVariable("KELDRIS_CONFIG_DIR", $null, "Machine")
        Write-Info "Removed KELDRIS_CONFIG_DIR environment variable"
    }

    # Remove install directory if empty
    if ((Test-Path $InstallDir) -and ((Get-ChildItem $InstallDir | Measure-Object).Count -eq 0)) {
        Remove-Item -Path $InstallDir -Force
        Write-Info "Removed empty install directory"
    }

    # Remove config and data if --Purge
    if ($Purge) {
        if (Test-Path $ConfigDir) {
            Write-Info "Removing configuration directory $ConfigDir..."
            Remove-Item -Path $ConfigDir -Recurse -Force
        }

        # Also clean up user home .keldris if it exists
        $userKeldris = Join-Path $env:USERPROFILE ".keldris"
        if (Test-Path $userKeldris) {
            Write-Info "Removing $userKeldris..."
            Remove-Item -Path $userKeldris -Recurse -Force
        }
    }
    else {
        if (Test-Path $ConfigDir) {
            Write-Warn "Configuration directory $ConfigDir was preserved."
            Write-Warn "Run with -Purge to remove all configuration and data."
        }
    }

    Write-Info "Keldris Agent has been uninstalled."
}

Main
