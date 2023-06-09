#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${SCRIPT_ROOT}"
ROOT_PATH=$(pwd)

IMPORT_ALIASES_PATH="${ROOT_PATH}/hack/.import-aliases"
INCLUDE_PATH="(${ROOT_PATH}/cmd|${ROOT_PATH}/test|${ROOT_PATH}/pkg)"
EXCLUDE_PATH="(${ROOT_PATH}/pkg/.*/mocks)"


ret=0
# We can't directly install preferredimports by `go install` due to the go.mod issue:
# go install k8s.io/kubernetes/cmd/preferredimports@v1.21.3: k8s.io/kubernetes@v1.21.3
#   The go.mod file for the module providing named packages contains one or
#   more replace directives. It must not contain directives that would cause
#   it to be interpreted differently than if it were the main module.
go run "${ROOT_PATH}/hack/tools/preferredimports/preferredimports.go" -import-aliases "${IMPORT_ALIASES_PATH}" -include-path "${INCLUDE_PATH}" -exclude-path "${EXCLUDE_PATH}" "${ROOT_PATH}" || ret=$?
if [[ $ret -ne 0 ]]; then
  echo "!!! Please see hack/.import-aliases for the preferred aliases for imports." >&2
  exit 1
fi
