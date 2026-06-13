# Instalador do jira-cli para Windows (PowerShell).
# Detecta arquitetura, baixa a ultima release do GitHub e instala no PATH.
# Uso: powershell -c "irm https://raw.githubusercontent.com/caiocesarps/jira-cli/main/install.ps1 | iex"

$ErrorActionPreference = 'Stop'

$Repo = 'caiocesarps/jira-cli'
$Binary = 'jira.exe'

function Write-Info { param($Message) Write-Host "[INFO] $Message" -ForegroundColor Green }
function Write-Warn { param($Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }
function Write-ErrorX { param($Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

# Detecta arquitetura
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'AMD64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'AMD64') {
    'amd64'
} elseif ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    'arm64'
} else {
    Write-ErrorX 'Arquitetura nao suportada.'
    exit 1
}

$OS = 'windows'

# Busca a ultima versao
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name
if (-not $Version) {
    Write-ErrorX 'Nao foi possivel determinar a ultima versao.'
    exit 1
}

Write-Info "Detectado: $OS/$Arch"
Write-Info "Ultima versao encontrada: $Version"

$Asset = "jira-cli-$($Version.TrimStart('v'))-$OS-$Arch.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Asset"
$ChecksumUrl = "https://github.com/$Repo/releases/download/$Version/checksums.txt"

$TmpDir = Join-Path $env:TEMP "jira-cli-install-$(Get-Random)"
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Write-Info "Baixando $Asset ..."
    Invoke-WebRequest -Uri $Url -OutFile (Join-Path $TmpDir $Asset) -UseBasicParsing
    Invoke-WebRequest -Uri $ChecksumUrl -OutFile (Join-Path $TmpDir 'checksums.txt') -UseBasicParsing

    # Verifica checksum
    $ChecksumLine = Get-Content (Join-Path $TmpDir 'checksums.txt') | Where-Object { $_ -match "^\S+\s+\*?$Asset" }
    if ($ChecksumLine) {
        $ExpectedHash = ($ChecksumLine -split '\s+')[0]
        $ActualHash = (Get-FileHash (Join-Path $TmpDir $Asset) -Algorithm SHA256).Hash.ToLower()
        if ($ExpectedHash -ne $ActualHash) {
            Write-ErrorX 'Falha na verificacao de checksum.'
            exit 1
        }
        Write-Info 'Checksum verificado.'
    } else {
        Write-Warn 'Checksum do asset nao encontrado; pulando verificacao.'
    }

    Write-Info 'Extraindo arquivo...'
    Expand-Archive -Path (Join-Path $TmpDir $Asset) -DestinationPath $TmpDir -Force

    $ExtractedBinary = Get-ChildItem -Path $TmpDir -Recurse -Filter $Binary | Select-Object -First 1
    if (-not $ExtractedBinary) {
        Write-ErrorX 'Binario nao encontrado apos extracao.'
        exit 1
    }

    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs' 'jira-cli'
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

    $Destination = Join-Path $InstallDir $Binary
    Move-Item -Path $ExtractedBinary.FullName -Destination $Destination -Force

    # Adiciona ao PATH do usuario se ainda nao estiver
    $UserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($UserPath -notlike "*$InstallDir*") {
        Write-Info "Adicionando $InstallDir ao PATH do usuario..."
        [Environment]::SetEnvironmentVariable('Path', "$UserPath;$InstallDir", 'User')
        $env:Path = "$env:Path;$InstallDir"
        Write-Warn 'Reinicione o terminal para garantir que o PATH foi atualizado.'
    }

    Write-Info 'Instalacao concluida!'
    & $Destination version
} finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
