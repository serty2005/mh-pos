#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

set -a
source tools/plane-mcp/.env
set +a

: "${PLANE_BASE_URL:?missing PLANE_BASE_URL in tools/plane-mcp/.env}"
: "${PLANE_WORKSPACE_SLUG:?missing PLANE_WORKSPACE_SLUG in tools/plane-mcp/.env}"
: "${PLANE_API_KEY:?missing PLANE_API_KEY in tools/plane-mcp/.env}"

exec uvx plane-mcp-server stdio
