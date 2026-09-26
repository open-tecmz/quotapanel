#!/usr/bin/env bash
# 更新 QuotaPanel 版本号（app.go / 前端各 package.json / test/screenshot.ts）
#
# 用法：
#   scripts/update-version.sh 0.2.5      # v 前缀可选
#   make update-version 0.2.5
#   make update-version VERSION=0.2.5

set -euo pipefail

if [ $# -lt 1 ] || [ -z "${1:-}" ]; then
  echo "用法：scripts/update-version.sh <version>   （如 0.2.5 或 v0.2.5）" >&2
  echo "      make update-version 0.2.5" >&2
  exit 1
fi
VERSION_ARG="$1"

# Go 侧版本保留 v 前缀，前端 npm 侧去掉 v 前缀
if [[ "$VERSION_ARG" == v* ]]; then
  VER_V="$VERSION_ARG"
  VER_PLAIN="${VERSION_ARG#v}"
else
  VER_V="v$VERSION_ARG"
  VER_PLAIN="$VERSION_ARG"
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# 当前版本以 app.go 中的 Version 为准
OLD_V="$(sed -n 's/.*Version:[[:space:]]*"\(v[^"]*\)".*/\1/p' app.go | head -1)"
if [ -z "$OLD_V" ]; then
  echo "错误：无法从 app.go 读取当前版本号" >&2
  exit 1
fi

esc() { printf '%s' "$1" | sed 's/\./\\./g'; }
OLD_V_ESC="$(esc "$OLD_V")"

echo "更新版本：${OLD_V} -> ${VER_V}"

# app.go：Version: "vX.Y.Z"
sed -i.bak -E "s|(Version:[[:space:]]*)\"$OLD_V_ESC\"|\1\"$VER_V\"|" app.go

# 前端各 package.json：仅替换包自身 version 字段，不影响依赖
PKG_FILES="frontend/package.json frontend/packages/ui/package.json frontend/packages/quota/package.json"
for f in $PKG_FILES; do
  [ -f "$f" ] || continue
  sed -i.bak -E "s|(\"version\":[[:space:]]*)\"[^\"]*\"|\1\"$VER_PLAIN\"|" "$f"
done

# test/screenshot.ts：mock 版本号（Pro 专属，Open 版可能不存在）
if [ -f test/screenshot.ts ]; then
  sed -i.bak "s|$OLD_V_ESC|$VER_V|g" test/screenshot.ts
fi

rm -f app.go.bak frontend/package.json.bak frontend/packages/ui/package.json.bak \
  frontend/packages/quota/package.json.bak test/screenshot.ts.bak

echo "✅ 版本已更新为 ${VER_V}（app.go / 前端 package.json / test/screenshot.ts）"
