#!/usr/bin/env bash
set -euo pipefail

: "${LICENSE_DEPLOY_HOST:?LICENSE_DEPLOY_HOST is required}"
: "${LICENSE_DEPLOY_USER:?LICENSE_DEPLOY_USER is required}"
: "${LICENSE_DEPLOY_KEY_PATH:?LICENSE_DEPLOY_KEY_PATH is required}"

port="${LICENSE_DEPLOY_PORT:-22}"
binary="${LICENSE_DEPLOY_BINARY:-license-server/license-api}"
release="${LICENSE_DEPLOY_RELEASE:-$(date -u +%Y%m%dT%H%M%SZ)-${GITHUB_SHA:-local}}"
remote_root="${LICENSE_DEPLOY_ROOT:-/opt/myhoreca/license-server}"
remote_release="$remote_root/releases/$release"

ssh_base=(ssh -i "$LICENSE_DEPLOY_KEY_PATH" -p "$port" -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes)
scp_base=(scp -i "$LICENSE_DEPLOY_KEY_PATH" -P "$port" -o IdentitiesOnly=yes -o StrictHostKeyChecking=yes)
target="$LICENSE_DEPLOY_USER@$LICENSE_DEPLOY_HOST"

"${ssh_base[@]}" "$target" "sudo install -d -m 0755 '$remote_release'"
"${scp_base[@]}" "$binary" "$target:/tmp/license-api.$release"
"${ssh_base[@]}" "$target" "sudo install -m 0755 /tmp/license-api.$release '$remote_release/license-api' && rm -f /tmp/license-api.$release && sudo ln -sfn '$remote_release' '$remote_root/current' && sudo systemctl restart license-api && sudo systemctl --no-pager --full status license-api"
