#!/bin/bash -eu

set -o pipefail

: "${OUT_DIR:?Must set OUT_DIR}"

TOP="$(readlink -f "$(dirname "$0")"/../../..)"
mkdir -p ${OUT_DIR}
export TMPDIR=$(cd ${OUT_DIR}; pwd)/tmp
mkdir -p ${TMPDIR}

"$TOP/build/soong/scripts/run-soong-tests-with-go-tools.sh"
"$TOP/build/soong/tests/run_integration_tests.sh"
