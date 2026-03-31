#!/bin/bash

# Default version if not provided
VERSION=${1:-"v1.0.0"}
OUT_DIR="release/${VERSION}"
mkdir -p "${OUT_DIR}"

# Targets
PLATFORMS=("windows/amd64" "windows/arm64" "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64")

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS="/" read -r OS ARCH <<< "${PLATFORM}"
    OUT_FILE="tiecli-${VERSION}-${OS}-${ARCH}"
    if [ "${OS}" == "windows" ]; then
        OUT_FILE="${OUT_FILE}.exe"
    fi

    echo "Building ${OS}/${ARCH}..."
    GOOS=${OS} GOARCH=${ARCH} go build -o "${OUT_DIR}/${OUT_FILE}" main.go

    if [ $? -eq 0 ]; then
        echo "Doneing ${OUT_FILE}"
        # Archive
        cd "${OUT_DIR}"
        if [ "${OS}" == "windows" ]; then
            zip "${OUT_FILE}.zip" "${OUT_FILE}" && rm "${OUT_FILE}"
        else
            tar -czf "${OUT_FILE}.tar.gz" "${OUT_FILE}" && rm "${OUT_FILE}"
        fi
        cd ../..
    else
        echo "Error: Build failed for ${OS}/${ARCH}"
    fi
done

echo "Release artifacts are located in ${OUT_DIR}"
