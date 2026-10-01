#!/bin/bash
# inis 单文件打包脚本（Linux / macOS）
#
# 作用：构建 Mellow 前端产物 → 拷到 theme/dist → 用 -tags embed 编译，
# 得到一个「自带主题」的可执行文件：部署时只要「二进制 + config 目录」，
# 不必再把主题文件分发到 public 目录，也不用额外配 nginx 反代静态资源。
#
# 用法：
#   bash build.sh                      # 按当前系统/架构编译，并内嵌前端
#   bash build.sh linux amd64          # 交叉编译到 Linux x86_64
#   bash build.sh linux arm64          # 交叉编译到 Linux ARM64（鲲鹏 / 树莓派）
#   bash build.sh linux amd64 skip     # 跳过前端构建（不内嵌主题，行为与老版本一致）
#
# 产物：./dist/inis_<os>_<arch>[.exe]
#
# 说明：
#   - 未安装 Node.js / 前端构建失败时，脚本会**继续编译但不内嵌主题**，
#     此时仍需自行把主题文件部署到 public 目录（与老版本部署方式一致）；
#   - 想要 UPX 压缩可自行在最后一步后执行：upx -6 --best --lzma dist/inis_linux_amd64
set -euo pipefail

cd "$(dirname "$0")"

GOOS_TARGET="${1:-$(go env GOOS)}"
GOARCH_TARGET="${2:-$(go env GOARCH)}"
SKIP_FRONT="${3:-}"

OUT_DIR="./dist"
OUT_NAME="inis_${GOOS_TARGET}_${GOARCH_TARGET}"
[ "$GOOS_TARGET" = "windows" ] && OUT_NAME="${OUT_NAME}.exe"

FRONT_DIR="Mellow"
EMBED_DIR="theme/dist"
TAGS=""

if [ "$SKIP_FRONT" = "skip" ]; then
  echo "[跳过] 按要求不重新构建前端"
elif command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
  echo "[构建] 前端（$FRONT_DIR）..."
  [ -d "$FRONT_DIR/node_modules" ] || (cd "$FRONT_DIR" && npm install --no-fund --no-audit)
  (cd "$FRONT_DIR" && npm run build)
else
  echo "[警告] 未安装 Node.js，跳过前端构建（本次不内嵌主题）"
fi

if [ -f "$FRONT_DIR/dist/index.html" ]; then
  echo "[打包] 拷贝前端产物到 $EMBED_DIR ..."
  rm -rf "$EMBED_DIR"
  mkdir -p "$EMBED_DIR"
  cp -R "$FRONT_DIR/dist/." "$EMBED_DIR/"
  TAGS="-tags embed"
  echo "[成功] 前端产物已就绪，编译时打进二进制"
else
  echo "[警告] 没找到 $FRONT_DIR/dist/index.html，本次编译不内嵌主题"
  echo "       部署时仍需把主题文件放进 public 目录"
fi

echo "[编译] $GOOS_TARGET/$GOARCH_TARGET ..."
mkdir -p "$OUT_DIR"
CGO_ENABLED=0 GOOS="$GOOS_TARGET" GOARCH="$GOARCH_TARGET" \
  go build $TAGS -ldflags "-s -w -buildid=" -trimpath -o "$OUT_DIR/$OUT_NAME" main.go

echo "[完成] $OUT_DIR/$OUT_NAME（$(du -h "$OUT_DIR/$OUT_NAME" | cut -f1)）"
