#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="${1:-$ROOT/dist/pos-edge-windows}"

rm -rf "$OUT"
mkdir -p "$OUT"

(
  cd "$ROOT/pos-ui-g"
  npm install
  VITE_POS_API_BASE=/api/v1 npm run build
)

for arch in amd64 386; do
  pkg="$OUT/windows-$arch"
  mkdir -p "$pkg/config" "$pkg/migrations" "$pkg/ui" "$pkg/webwallpaper"

  (
    cd "$ROOT/pos-backend"
    CGO_ENABLED=0 GOOS=windows GOARCH="$arch" go build -trimpath -ldflags="-s -w" -o "$pkg/pos-edge.exe" ./cmd/pos-edge
  )

  cp "$ROOT/pos-backend/config/pos-edge.windows.json" "$pkg/config/pos-edge.json"
  cp -R "$ROOT/pos-backend/migrations/sqlite" "$pkg/migrations/sqlite"
  cp -R "$ROOT/pos-ui-g/dist" "$pkg/ui/pos-ui"
  cp "$ROOT/docs/deployment/POS-EDGE-WINDOWS.md" "$pkg/README.POS-EDGE-WINDOWS.md"
  cat > "$pkg/webwallpaper/config.pos-edge.example.json" <<'JSON'
{
  "URL": "http://127.0.0.1:8080/",
  "Monitors": [],
  "Audio": {
    "ID": "",
    "Name": "",
    "Active": false
  }
}
JSON
done

printf 'POS Edge Windows packages written to %s\n' "$OUT"
