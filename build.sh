#!/bin/bash

VERSION=${1:-dev}
BINARY_NAME="jms_aid"
OUTPUT_DIR="target"
TAR_PREFIX="${BINARY_NAME}_${VERSION}"

# 清理并创建目录
rm -rf "${OUTPUT_DIR}" && mkdir -p "${OUTPUT_DIR}"

# 通用构建函数
build_for_arch() {
    local arch=$1
    local tmp_bin="${BINARY_NAME}_${arch}"
    local output="${OUTPUT_DIR}/${TAR_PREFIX}_${arch}"

    GOOS=linux GOARCH="${arch}" go build -o "${tmp_bin}" .
    chmod +x "${tmp_bin}"
    mkdir -p "${output}"
    mv "${tmp_bin}" "${output}/${BINARY_NAME}"
    tar -C "${OUTPUT_DIR}" -czf "${output}.tar.gz" "${TAR_PREFIX}_${arch}"
    rm -rf "${output}"  # 清理临时目录
}

# 并行构建（需 Bash 4.3+）
build_for_arch amd64 &
build_for_arch arm64 &
wait

echo "构建完成: ${OUTPUT_DIR}/${TAR_PREFIX}_*.tar.gz"