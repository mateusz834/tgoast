#/usr/bin/env bash
set -e
set -x
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd "$SCRIPT_DIR/.."
find . -name '*.go' -not -path '*/testdata/*' -exec goimports -w {} +
