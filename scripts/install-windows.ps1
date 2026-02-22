#Requires -RunAsAdministrator
<#
.SYNOPSIS
    Keldris Agent Installer for Windows

.DESCRIPTION
    Downloads the Keldris Agent binary, installs it to Program Files,
    and creates a Windows Service for automatic startup.

.PARAMETER Action
    The action to perform: Install or Uninstall

.PARAMETER Version
    The version to install. Defaults to 'latest'.

.PARAMETER DownloadUrl
    Base URL for downloading the agent binary.

.EXAMPLE
    .\install-windows.ps1 -Action Install

.EXAMPLE
    .\install-windows.ps1 -Action Uninstall
#>

param(
    [Parameter(Mandatory = $false)]
    [ValidateSet("Install", "Uninstall")]
    [string]$Action = "Install",

    [Parameter(Mandatory = $false)]
    [string]$Version = "latest",

    [Parameter(Mandatory = $false)]
    [string]$DownloadUrl = "https://github.com/MacJediWizard/keldris/releases/latest/download"
    [string]$DownloadUrl = "https://releases.keldris.io/agent"
)

# Configuration
$ServiceName = "KeldrisAgent"
$ServiceDisplayName = "Keldris Backup Agent"
$ServiceDescription = "Keldris backup agent for automated system backups"
$BinaryName = "keldris-agent.exe"
$InstallDir = "$env:ProgramFiles\Keldris"
$ConfigDir = "$env:ProgramData\Keldris"
$LogFile = "$ConfigDir\install.log"

# Colors for output
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

# Check if running as Administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Detect system architecture
function Get-SystemArch {
    if ([Environment]::Is64BitOperatingSystem) {
        return "amd64"
    }
    else {
        Write-Error "32-bit Windows is not supported"
        exit 1
    }
}

# Download the binary
function Get-AgentBinary {
    param([string]$Arch)

    $downloadFullUrl = "$DownloadUrl/keldris-agent-windows-$Arch.exe"
    $downloadFullUrl = "$DownloadUrl/$Version/keldris-agent-windows-$Arch.exe"
    $tempFile = Join-Path $env:TEMP $BinaryName

    Write-Info "Downloading Keldris Agent ($Version, windows/$Arch)..."
    Write-Info "URL: $downloadFullUrl"

    try {
        # Use TLS 1.2 or higher
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

        $webClient = New-Object System.Net.WebClient
        $webClient.DownloadFile($downloadFullUrl, $tempFile)

        Write-Info "Download complete"
        return $tempFile
    }
    catch {
        Write-Error "Failed to download binary: $_"
        exit 1
    }
}

# Install the binary
function Install-AgentBinary {
    param([string]$TempFile)

    Write-Info "Installing binary to $InstallDir..."

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $installPath = Join-Path $InstallDir $BinaryName

    # Stop service if running
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service -and $service.Status -eq "Running") {
        Write-Info "Stopping existing service..."
        Stop-Service -Name $ServiceName -Force
        Start-Sleep -Seconds 2
    }

    # Copy binary
    Copy-Item -Path $TempFile -Destination $installPath -Force
    Remove-Item -Path $TempFile -Force -ErrorAction SilentlyContinue

    Write-Info "Binary installed to $installPath"
    return $installPath
}

# Create configuration directory
function New-ConfigDirectory {
    Write-Info "Creating configuration directory..."

    if (-not (Test-Path $ConfigDir)) {
        New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
    }

    # Set permissions (restrict to Administrators and SYSTEM)
    $acl = Get-Acl $ConfigDir
    $acl.SetAccessRuleProtection($true, $false)

    $adminRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "BUILTIN\Administrators",
        "FullControl",
        "ContainerInherit,ObjectInherit",
        "None",
        "Allow"
    )
    $systemRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
        "NT AUTHORITY\SYSTEM",
        "FullControl",
        "ContainerInherit,ObjectInherit",
        "None",
        "Allow"
    )

    $acl.AddAccessRule($adminRule)
    $acl.AddAccessRule($systemRule)
    Set-Acl -Path $ConfigDir -AclObject $acl

    Write-Info "Configuration directory created at $ConfigDir"
}

# Create Windows Service
function New-AgentService {
    param([string]$BinaryPath)

    Write-Info "Creating Windows Service..."

    # Check if service already exists
    $existingService = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existingService) {
        Write-Info "Service already exists. Updating..."
        # Stop the service first
        if ($existingService.Status -eq "Running") {
            Stop-Service -Name $ServiceName -Force
            Start-Sleep -Seconds 2
        }
        # Remove old service
        sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Seconds 2
    }

    # Create the service
    $serviceBinPath = "`"$BinaryPath`" daemon"

    $result = sc.exe create $ServiceName binPath= $serviceBinPath start= auto DisplayName= $ServiceDisplayName
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create service: $result"
        exit 1
    }

    # Set service description
    sc.exe description $ServiceName $ServiceDescription | Out-Null

    # Configure service recovery options (restart on failure)
    sc.exe failure $ServiceName reset= 86400 actions= restart/10000/restart/30000/restart/60000 | Out-Null

    # Set service to delayed auto-start
    sc.exe config $ServiceName start= delayed-auto | Out-Null

    Write-Info "Windows Service created: $ServiceDisplayName"
}

# Start the service
function Start-AgentService {
    Write-Info "Starting service..."

    try {
        Start-Service -Name $ServiceName
        Start-Sleep -Seconds 2

        $service = Get-Service -Name $ServiceName
        if ($service.Status -eq "Running") {
            Write-Info "Service started successfully"
        }
        else {
            Write-Warn "Service may not have started correctly. Status: $($service.Status)"
        }
    }
    catch {
        Write-Warn "Failed to start service: $_"
        Write-Warn "You may need to configure the agent first and start the service manually"
    }
}

# Add to PATH
function Add-ToPath {
    Write-Info "Adding install directory to PATH..."

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    if ($currentPath -notlike "*$InstallDir*") {
        $newPath = "$currentPath;$InstallDir"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
        Write-Info "Added $InstallDir to system PATH"
        Write-Warn "You may need to restart your terminal for PATH changes to take effect"
    }
    else {
        Write-Info "Install directory already in PATH"
    }
}

# Print instructions
function Show-Instructions {
    Write-Host ""
    Write-Host "==============================================" -ForegroundColor Cyan
    Write-Host "  Keldris Agent Installation Complete" -ForegroundColor Cyan
    Write-Host "==============================================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Next steps:"
    Write-Host "  1. Register the agent with your Keldris server:"
    Write-Host "     keldris-agent register --server https://your-server.com" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  2. Check agent status:"
    Write-Host "     keldris-agent status" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Service management (PowerShell as Administrator):"
    Write-Host "  Start:   Start-Service $ServiceName"
    Write-Host "  Stop:    Stop-Service $ServiceName"
    Write-Host "  Status:  Get-Service $ServiceName"
    Write-Host "  Logs:    Get-EventLog -LogName Application -Source $ServiceName"
    Write-Host ""
    Write-Host "Install directory: $InstallDir"
    Write-Host "Config directory:  $ConfigDir"
    Write-Host ""
}

# Uninstall
function Uninstall-Agent {
    Write-Info "Uninstalling Keldris Agent..."

    # Stop and remove service
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq "Running") {
            Write-Info "Stopping service..."
            Stop-Service -Name $ServiceName -Force
            Start-Sleep -Seconds 2
        }
        Write-Info "Removing service..."
        sc.exe delete $ServiceName | Out-Null
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
        Write-Info "Removed from PATH"
    }

    # Remove install directory if empty
    if ((Test-Path $InstallDir) -and ((Get-ChildItem $InstallDir | Measure-Object).Count -eq 0)) {
        Remove-Item -Path $InstallDir -Force
    }

    Write-Info "Uninstall complete"
    Write-Warn "Configuration directory $ConfigDir was not removed. Delete manually if needed."
}

# Main
function Main {
    if (-not (Test-Administrator)) {
        Write-Error "This script must be run as Administrator"
        Write-Host "Right-click PowerShell and select 'Run as Administrator'"
        exit 1
    }

    switch ($Action) {
        "Install" {
            Write-Info "Starting Keldris Agent installation..."

            $arch = Get-SystemArch
            Write-Info "Detected architecture: $arch"

            $tempFile = Get-AgentBinary -Arch $arch
            $binaryPath = Install-AgentBinary -TempFile $tempFile
            New-ConfigDirectory
            New-AgentService -BinaryPath $binaryPath
            Add-ToPath
            Start-AgentService
            Show-Instructions
        }
        "Uninstall" {
            Uninstall-Agent
        }
    }
}

Main
