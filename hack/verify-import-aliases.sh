#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${SCRIPT_ROOT}"
ROOT_PATH=$(pwd)

IMPORT_ALIASES_PATH="${ROOT_PATH}/hack/.import-aliases"

ret=0
go run "k8s.io/kubernetes/cmd/preferredimports" -import-aliases "${IMPORT_ALIASES_PATH}" "${ROOT_PATH}" || ret=$?
if [[ $ret -ne 0 ]]; then
  echo "!!! Please see hack/.import-aliases for the preferred aliases for imports." >&2
  exit 1
fi
