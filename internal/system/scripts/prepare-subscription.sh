#!/bin/sh
set -eu

if [ "$#" -lt 2 ]; then
  printf '%s\n' "usage: prepare-subscription.sh <url> <output-dir> [timeout] [user-agent] [max-bytes]" >&2
  exit 2
fi

URL="$1"
OUT_DIR="$2"
TIMEOUT="${3:-15}"
USER_AGENT="${4:-clashctl/dev}"
MAX_BYTES="${5:-20971520}"

# Validate URL format (only allow http/https)
case "$URL" in
  http://*|https://*)
    # OK
    ;;
  *)
    printf '%s\n' "error: only http/https URLs are allowed" >&2
    exit 1
    ;;
esac

# Reject URLs with shell metacharacters
case "$URL" in
  *';'*|*'|'*|*'`'*|*'$('*|*'&&'*|*'||'*)
    printf '%s\n' "error: URL contains invalid characters" >&2
    exit 1
    ;;
esac

CONTENT_FILE="$OUT_DIR/subscription.txt"
INFO_FILE="$OUT_DIR/subscription.info"

mkdir -p "$OUT_DIR"

unset http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY no_proxy NO_PROXY

FETCHER=""
if command -v curl >/dev/null 2>&1; then
  FETCHER="curl"
  # Use -- to prevent URL from being interpreted as options
  curl --noproxy '*' --connect-timeout "$TIMEOUT" --max-time "$TIMEOUT" --max-filesize "$MAX_BYTES" -fsSL -A "$USER_AGENT" -- "$URL" -o "$CONTENT_FILE"
elif command -v wget >/dev/null 2>&1; then
  FETCHER="wget"
  STATUS_FILE="$OUT_DIR/fetch.status"
  READ_LIMIT=$((MAX_BYTES + 1))
  rm -f "$STATUS_FILE"
  (
    wget --no-proxy -T "$TIMEOUT" -U "$USER_AGENT" -qO - -- "$URL"
    printf '%s' "$?" > "$STATUS_FILE"
  ) | head -c "$READ_LIMIT" > "$CONTENT_FILE"
  STATUS="$(cat "$STATUS_FILE" 2>/dev/null || printf '1')"
  if [ "$STATUS" -ne 0 ]; then
    printf '%s\n' "wget download failed" >&2
    exit 1
  fi
else
  printf '%s\n' "curl and wget are not available" >&2
  exit 1
fi

BYTES=$(wc -c < "$CONTENT_FILE" | tr -d ' ')
if [ "$BYTES" -gt "$MAX_BYTES" ]; then
  printf '%s\n' "subscription content exceeds size limit" >&2
  exit 1
fi

{
  printf 'url=%s\n' "$URL"
  printf 'fetcher=%s\n' "$FETCHER"
  printf 'bytes=%s\n' "$BYTES"
} > "$INFO_FILE"

printf '%s\n' "$CONTENT_FILE"
