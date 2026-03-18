#!/bin/bash
# clashctl + Mihomo 一键安装脚本
# Usage: curl -sL https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh | sudo bash
set -euo pipefail

# ─── Config ───
INSTALL_DIR="/usr/local/bin"
CLASHCTL_REPO="DUcotd/clashctl"
MIHOMO_REPO="MetaCubeX/mihomo"
TIMEOUT=30
MAX_RETRIES=3

# ─── Colors ───
if [ -t 1 ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    CYAN='\033[0;36m'; BOLD='\033[1m'; DIM='\033[2m'; RESET='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; CYAN=''; BOLD=''; DIM=''; RESET=''
fi

# ─── Helpers ───
info()  { printf "  ${CYAN}▸${RESET} %s\n" "$*"; }
ok()    { printf "  ${GREEN}✓${RESET} %s\n" "$*"; }
warn()  { printf "  ${YELLOW}⚠${RESET}  %s\n" "$*"; }
err()   { printf "  ${RED}✗${RESET} %s\n" "$*" >&2; }
die()   { err "$@"; exit 1; }

# ─── Args ───
SKIP_MIHOMO=false
SKIP_CLASHCTL=false
CLASHCTL_VERSION="latest"
MIHOMO_VERSION="latest"
YES=false

usage() {
    cat <<EOF
${BOLD}clashctl 安装脚本${RESET}

${BOLD}用法:${RESET}
  install.sh [选项]

${BOLD}选项:${RESET}
  --clashctl-only     只安装 clashctl
  --mihomo-only       只安装 Mihomo
  --clashctl-version  指定 clashctl 版本 (默认: latest)
  --mihomo-version    指定 Mihomo 版本 (默认: latest)
  --install-dir       安装目录 (默认: /usr/local/bin)
  -y, --yes           跳过确认
  -h, --help          显示帮助

${BOLD}示例:${RESET}
  # 一键安装全部
  curl -sL https://raw.githubusercontent.com/DUcotd/clashctl/main/install.sh | sudo bash

  # 只安装 clashctl
  sudo bash install.sh --clashctl-only

  # 指定版本
  sudo bash install.sh --clashctl-version v2.3.0 --mihomo-version v1.19.0
EOF
    exit 0
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --clashctl-only)    SKIP_MIHOMO=true ;;
        --mihomo-only)      SKIP_CLASHCTL=true ;;
        --clashctl-version) CLASHCTL_VERSION="$2"; shift ;;
        --mihomo-version)   MIHOMO_VERSION="$2"; shift ;;
        --install-dir)      INSTALL_DIR="$2"; shift ;;
        -y|--yes)           YES=true ;;
        -h|--help)          usage ;;
        *)                  die "未知选项: $1 (用 -h 查看帮助)" ;;
    esac
    shift
done

# ─── Pre-flight ───
[ "$EUID" -eq 0 ] || die "请使用 sudo 运行: sudo bash $0"
command -v curl &>/dev/null || die "需要 curl，请先安装"

OS=$(uname -s)
ARCH=$(uname -m)
[ "$OS" = "Linux" ] || die "暂仅支持 Linux，当前: $OS"

case "$ARCH" in
    x86_64|amd64)   GOARCH="amd64" ;;
    aarch64|arm64)  GOARCH="arm64" ;;
    armv7l|armv6l)  GOARCH="armv7" ;;
    *)              die "不支持的架构: $ARCH" ;;
esac

# ─── Download helper with retry ───
download() {
    local url="$1" output="$2"
    local attempt=0
    while [ $attempt -lt $MAX_RETRIES ]; do
        if [ -n "$output" ]; then
            if curl -sfL --connect-timeout "$TIMEOUT" --retry 2 "$url" -o "$output"; then
                return 0
            fi
        else
            if curl -sfL --connect-timeout "$TIMEOUT" --retry 2 "$url"; then
                return 0
            fi
        fi
        attempt=$((attempt + 1))
        [ $attempt -lt $MAX_RETRIES ] && warn "下载失败，重试 ($attempt/$MAX_RETRIES)..." && sleep 2
    done
    return 1
}

# ─── JSON helper ───
json_field() {
    python3 -c "import sys,json; print(json.load(sys.stdin)$1)" 2>/dev/null
}

# ─── Install clashctl ───
install_clashctl() {
    echo ""
    printf "  ${BOLD}[1/2] 安装 clashctl${RESET}\n"

    # Check local binary first
    if [ -f "./clashctl-linux-amd64" ]; then
        info "检测到本地文件，直接安装"
        cp ./clashctl-linux-amd64 "$INSTALL_DIR/clashctl"
        chmod +x "$INSTALL_DIR/clashctl"
        ok "clashctl → $INSTALL_DIR/clashctl"
        return
    fi

    # Resolve version
    if [ "$CLASHCTL_VERSION" = "latest" ]; then
        info "获取最新版本..."
        local tag
        tag=$(download "https://api.github.com/repos/$CLASHCTL_REPO/releases/latest" | json_field "['tag_name']") || true
        CLASHCTL_VERSION="${tag:-latest}"
    fi

    local url
    if [ "$CLASHCTL_VERSION" = "latest" ]; then
        url="https://github.com/$CLASHCTL_REPO/releases/latest/download/clashctl-linux-${GOARCH}"
    else
        url="https://github.com/$CLASHCTL_REPO/releases/download/$CLASHCTL_VERSION/clashctl-linux-${GOARCH}"
    fi

    info "下载 ${DIM}$CLASHCTL_VERSION${RESET}..."
    download "$url" "$INSTALL_DIR/clashctl" || die "clashctl 下载失败: $url"
    chmod +x "$INSTALL_DIR/clashctl"
    ok "clashctl → $INSTALL_DIR/clashctl"
}

# ─── Install mihomo ───
install_mihomo() {
    echo ""
    printf "  ${BOLD}[2/2] 安装 Mihomo${RESET}\n"

    # Already installed?
    if command -v mihomo &>/dev/null; then
        local ver
        ver=$(mihomo -v 2>/dev/null | head -1 || echo "unknown")
        if [ "$MIHOMO_VERSION" = "latest" ]; then
            ok "已安装: $ver，跳过 (用 --mihomo-version 强制覆盖)"
            return
        else
            info "已安装 $ver，将覆盖为 $MIHOMO_VERSION"
        fi
    fi

    # Resolve version & URL
    local release_json
    if [ "$MIHOMO_VERSION" = "latest" ]; then
        info "获取最新版本..."
        release_json=$(download "https://api.github.com/repos/$MIHOMO_REPO/releases/latest") || die "无法获取 Mihomo 版本信息"
        MIHOMO_VERSION=$(echo "$release_json" | json_field "['tag_name']")
    else
        release_json=$(download "https://api.github.com/repos/$MIHOMO_REPO/releases/tags/$MIHOMO_VERSION") || die "无法获取 Mihomo $MIHOMO_VERSION"
    fi

    info "版本: ${DIM}$MIHOMO_VERSION${RESET}"

    # Find download URL - prefer .gz, then plain binary
    local mihomo_url
    mihomo_url=$(echo "$release_json" | python3 -c "
import sys, json
assets = json.load(sys.stdin).get('assets', [])
arch = '$GOARCH'
# Priority: .gz > plain binary (skip .deb, .rpm, .zst, .pkg.tar)
candidates = [a for a in assets if arch in a['name'] and not any(x in a['name'] for x in ['.deb', '.rpm', '.zst', '.pkg.tar'])]
if candidates:
    # Prefer .gz first
    gz = [a for a in candidates if a['name'].endswith('.gz')]
    print(gz[0]['browser_download_url'] if gz else candidates[0]['browser_download_url'])
" 2>/dev/null) || true

    [ -n "$mihomo_url" ] || die "找不到架构匹配的 Mihomo 二进制 ($GOARCH)"

    info "下载中 ${DIM}$(basename "$mihomo_url")${RESET}..."

    if [[ "$mihomo_url" == *.gz ]]; then
        download "$mihomo_url" - | gzip -d > "$INSTALL_DIR/mihomo" 2>/dev/null \
            || die "Mihomo 下载/解压失败"
    else
        download "$mihomo_url" "$INSTALL_DIR/mihomo" || die "Mihomo 下载失败"
    fi

    chmod +x "$INSTALL_DIR/mihomo"
    ok "mihomo → $INSTALL_DIR/mihomo ($MIHOMO_VERSION)"
}

# ─── Main ───
echo ""
printf "  ${BOLD}📦 clashctl 安装程序${RESET}\n"
printf "  ${DIM}架构: $OS/$GOARCH | 安装目录: $INSTALL_DIR${RESET}\n"

$SKIP_CLASHCTL || install_clashctl
$SKIP_MIHOMO   || install_mihomo

# ─── Done ───
echo ""
printf "  ${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}\n"
printf "  ${GREEN}✅ 安装完成！${RESET}\n"
echo ""
printf "  ${BOLD}开始使用：${RESET}\n"
printf "    ${CYAN}sudo clashctl init${RESET}    # 交互式配置向导\n"
printf "    ${CYAN}sudo clashctl doctor${RESET}  # 环境自检\n"
echo ""

# Show version
"$INSTALL_DIR/clashctl" version 2>/dev/null || true
