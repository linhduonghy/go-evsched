#!/usr/bin/env sh
set -e

TARGET_OS=${TARGET_OS:-linux}
TARGET_ARCH=${TARGET_ARCH:-amd64}
OUTPUT_DIR=${OUTPUT_DIR:-bin}
APP_NAME=${APP_NAME:-evsched}

EXTENSION=""
if [ "$TARGET_OS" = "windows" ]; then
  EXTENSION=".exe"
fi

TARGET_PATH="${OUTPUT_DIR}/${TARGET_OS}-${TARGET_ARCH}/${APP_NAME}${EXTENSION}"

mkdir -p "${OUTPUT_DIR}/${TARGET_OS}-${TARGET_ARCH}"

CGO_ENABLED=0 GOOS=$TARGET_OS GOARCH=$TARGET_ARCH go build \
  -trimpath \
  -ldflags="-s -w" \
  -o "${TARGET_PATH}" \
  ./cmd/server

echo "Built ${TARGET_PATH}"