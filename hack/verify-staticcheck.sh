#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
cd "${REPO_ROOT}"
source "hack/util.sh"

GOLANGCI_LINT_PKG="github.com/golangci/golangci-lint/cmd/golangci-lint"
GOLANGCI_LINT_VER="v1.50.1"
LINTER="golangci-lint"

which ${LINTER} || util::install_tools ${GOLANGCI_LINT_PKG} ${GOLANGCI_LINT_VER}

${LINTER} --version

if ${LINTER} run --timeout=10m; then
  echo 'Congratulations!  All Go source files have passed staticcheck.'
else
  echo # print one empty line, separate from warning messages.
  echo 'Please review the above warnings.'
  echo 'If the above warnings do not make sense, feel free to file an issue.'
  exit 1
fi


go install golang.org/x/tools/cmd/goimports@latest