#!/usr/bin/env bash
#
# Instalador multiplataforma do jira-cli.
# Detecta SO e arquitetura, baixa a ultima release do GitHub e instala no PATH.
# Uso: curl -sSL https://raw.githubusercontent.com/caiocesarps/jira-cli/main/install.sh | bash
#
set -euo pipefail

REPO="caiocesarps/jira-cli"
BINARY="jira"

# Cores
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Detecta SO
 detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux*)  echo "linux" ;;
        darwin*) echo "darwin" ;;
        mingw*|cygwin*|msys*|windowsnt*)
            echo "windows"
            ;;
        *)
            log_error "Sistema operacional nao suportado: $os"
            exit 1
            ;;
    esac
}

# Detecta arquitetura
 detect_arch() {
    local arch
    arch="$(uname -m | tr '[:upper:]' '[:lower:]')"
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        armv7l|armv7) echo "arm" ;;
        *)
            log_error "Arquitetura nao suportada: $arch"
            exit 1
            ;;
    esac
}

# Verifica se um comando existe
 command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Escolhe o diretorio de instalacao
 choose_install_dir() {
    local os="$1"
    if [ "$os" = "windows" ]; then
        # Preferencia: %USERPROFILE%\bin ou %LOCALAPPDATA%\Programs\jira-cli
        if [ -n "${LOCALAPPDATA:-}" ]; then
            echo "$LOCALAPPDATA/Programs/jira-cli"
        else
            echo "$HOME/bin"
        fi
        return
    fi

    # Unix: tenta /usr/local/bin, depois ~/.local/bin
    if [ -w "/usr/local/bin" ] 2>/dev/null || [ "$EUID" -eq 0 ]; then
        echo "/usr/local/bin"
    else
        echo "$HOME/.local/bin"
    fi
}

# Adiciona o diretorio ao PATH no shell profile
 ensure_path_unix() {
    local dir="$1"
    local shell_rc

    case "${SHELL##*/}" in
        bash) shell_rc="$HOME/.bashrc" ;;
        zsh)  shell_rc="$HOME/.zshrc" ;;
        fish) shell_rc="$HOME/.config/fish/config.fish" ;;
        *)    shell_rc="$HOME/.profile" ;;
    esac

    if [ -f "$shell_rc" ] && ! grep -q "$dir" "$shell_rc" 2>/dev/null; then
        log_info "Adicionando $dir ao PATH em $shell_rc"
        echo "export PATH=\"$dir:\$PATH\"" >> "$shell_rc"
        log_warn "Execute 'source $shell_rc' ou reinicie o terminal para atualizar o PATH."
    fi
}

main() {
    local os arch version asset ext url tmpdir install_dir target_name checksum_url

    os="$(detect_os)"
    arch="$(detect_arch)"

    log_info "Detectado: $os/$arch"

    # Busca a ultima versao
    if command_exists curl; then
        version="$(curl -sSL "https://api.github.com/repos/$REPO/releases/latest" | grep -o '"tag_name": "[^"]*"' | head -n 1 | sed 's/"tag_name": "//;s/"$//')"
    elif command_exists wget; then
        version="$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep -o '"tag_name": "[^"]*"' | head -n 1 | sed 's/"tag_name": "//;s/"$//')"
    else
        log_error "curl ou wget sao necessarios para continuar."
        exit 1
    fi

    if [ -z "$version" ]; then
        log_error "Nao foi possivel determinar a ultima versao."
        exit 1
    fi

    log_info "Ultima versao encontrada: $version"

    # Monta nome do asset
    if [ "$os" = "windows" ]; then
        ext="zip"
        target_name="${BINARY}.exe"
    else
        ext="tar.gz"
        target_name="$BINARY"
    fi

    asset="jira-cli-${version#v}-${os}-${arch}.${ext}"
    url="https://github.com/$REPO/releases/download/$version/$asset"
    checksum_url="https://github.com/$REPO/releases/download/$version/checksums.txt"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "${tmpdir:-}"' EXIT

    log_info "Baixando $asset ..."
    if command_exists curl; then
        curl -sSL -o "$tmpdir/$asset" "$url"
        curl -sSL -o "$tmpdir/checksums.txt" "$checksum_url"
    else
        wget -q -O "$tmpdir/$asset" "$url"
        wget -q -O "$tmpdir/checksums.txt" "$checksum_url"
    fi

    # Verifica checksum
    if command_exists sha256sum; then
        (
            cd "$tmpdir"
            grep "  $asset$" checksums.txt | sha256sum -c - >/dev/null 2>&1 || {
                log_error "Falha na verificacao de checksum."
                exit 1
            }
        )
        log_info "Checksum verificado."
    else
        log_warn "sha256sum nao encontrado; pulando verificacao de checksum."
    fi

    # Extrai
    log_info "Extraindo $asset ..."
    if [ "$ext" = "zip" ]; then
        unzip -q "$tmpdir/$asset" -d "$tmpdir/extract"
    else
        mkdir -p "$tmpdir/extract"
        tar -xzf "$tmpdir/$asset" -C "$tmpdir/extract"
    fi

    # Encontra o binario extraido
    local extracted_binary
    extracted_binary="$(find "$tmpdir/extract" -type f \( -name "$BINARY" -o -name "$BINARY.exe" \) | head -n 1)"
    if [ -z "$extracted_binary" ]; then
        log_error "Binario nao encontrado apos extracao."
        exit 1
    fi

    install_dir="$(choose_install_dir "$os")"
    mkdir -p "$install_dir"

    log_info "Instalando em $install_dir/$target_name ..."
    if [ "$os" != "windows" ] && [ ! -w "$install_dir" ]; then
        log_warn "Sem permissao de escrita em $install_dir. Tentando com sudo..."
        sudo mv "$extracted_binary" "$install_dir/$target_name"
        sudo chmod +x "$install_dir/$target_name"
    else
        mv "$extracted_binary" "$install_dir/$target_name"
        if [ "$os" != "windows" ]; then
            chmod +x "$install_dir/$target_name"
        fi
    fi

    # Unix: garante que o diretorio esteja no PATH
    if [ "$os" != "windows" ]; then
        ensure_path_unix "$install_dir"
    fi

    log_info "Instalacao concluida!"
    if command_exists "$install_dir/$target_name"; then
        "$install_dir/$target_name" version
    else
        log_warn "$install_dir pode nao estar no PATH."
    fi
}

main "$@"
