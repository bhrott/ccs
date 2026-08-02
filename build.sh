#!/usr/bin/env bash
#
# Builds the ccs binary.
#
#   ./build.sh              # builds ./bin/ccs
#   ./build.sh --install    # builds and copies it to ~/.local/bin
#   ./build.sh -o /tmp      # builds into another folder
#
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIR="$PROJECT_DIR/bin"
INSTALL_DIR="${CCS_INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="ccs"
install=false

usage() {
    cat <<EOF
Usage: ./build.sh [options]

Options:
  -o, --output DIR   Folder where the binary is written (default: ./bin)
  -i, --install      Also copy the binary to $INSTALL_DIR
  -h, --help         Show this help

Environment:
  CCS_INSTALL_DIR    Folder used by --install (default: ~/.local/bin)
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -o|--output)
            [[ $# -ge 2 ]] || { echo "error: $1 requires a folder" >&2; exit 1; }
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -i|--install)
            install=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "error: unknown option: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if ! command -v go >/dev/null 2>&1; then
    echo "error: go is not installed or not in PATH" >&2
    exit 1
fi

cd "$PROJECT_DIR"

version="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
binary_path="$OUTPUT_DIR/$BINARY_NAME"

mkdir -p "$OUTPUT_DIR"

echo "==> building $BINARY_NAME $version"
go build -trimpath -ldflags "-s -w -X github.com/bhrott/ccs/internal/cli.Version=$version" -o "$binary_path" .
echo "==> built $binary_path"

if [[ "$install" == true ]]; then
    mkdir -p "$INSTALL_DIR"
    cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    echo "==> installed $INSTALL_DIR/$BINARY_NAME"

    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *) echo "    note: $INSTALL_DIR is not in your PATH, add it to run '$BINARY_NAME' from anywhere" ;;
    esac
fi

echo
echo "Run it with:"
if [[ "$install" == true ]]; then
    echo "  $BINARY_NAME ls"
else
    echo "  $binary_path ls"
fi
echo
echo "Point it at your sheets with:"
echo "  export CHEAT_SHEETS_FILE_PATH=$PROJECT_DIR/cheat-sheets.yaml"
