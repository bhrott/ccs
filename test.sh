#!/usr/bin/env bash
#
# Runs the checks of the project: formatting, vet and the unit tests.
#
#   ./test.sh              # gofmt + go vet + go test
#   ./test.sh -v           # same, with the verbose test output
#   ./test.sh --cover      # also writes a coverage report
#
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COVERAGE_FILE="$PROJECT_DIR/coverage.out"
verbose=false
cover=false
race=false

usage() {
    cat <<EOF
Usage: ./test.sh [options]

Options:
  -v, --verbose   Show the output of every test
  -c, --cover     Write coverage.out and print the total coverage
  -r, --race      Run the tests with the race detector
  -h, --help      Show this help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -v|--verbose) verbose=true; shift ;;
        -c|--cover) cover=true; shift ;;
        -r|--race) race=true; shift ;;
        -h|--help) usage; exit 0 ;;
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

echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
    echo "error: these files are not formatted, run 'gofmt -w .':" >&2
    echo "$unformatted" >&2
    exit 1
fi

echo "==> go vet"
go vet ./...

echo "==> go test"
test_args=(./...)
[[ "$verbose" == true ]] && test_args+=(-v)
[[ "$race" == true ]] && test_args+=(-race)
[[ "$cover" == true ]] && test_args+=(-coverprofile="$COVERAGE_FILE" -covermode=atomic)

go test "${test_args[@]}"

if [[ "$cover" == true ]]; then
    echo "==> coverage"
    go tool cover -func="$COVERAGE_FILE" | tail -1
    echo "    report: $COVERAGE_FILE (open with: go tool cover -html=$COVERAGE_FILE)"
fi

echo
echo "all checks passed"
