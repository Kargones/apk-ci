#!/bin/bash
# Запуск интеграционного теста nr-convert-pipeline (3 отдельных шага).
# Эмулирует Gitea Actions workflow: nr-convert → nr-git2store → nr-extension-publish
#
# Использование:
#   ./scripts/run-pipeline-integration.sh
#   GITEA_REPO=test/SURV ./scripts/run-pipeline-integration.sh
#   GITEA_TOKEN=xxx GITEA_REPO=test/TOIR3 GITEA_REF_NAME=v18 ./scripts/run-pipeline-integration.sh

set -euo pipefail

: "${GITEA_TOKEN:=e0452e72c27392799fd34f88da9546a1af509947}"
: "${GITEA_URL:=https://git.apkholding.ru}"
: "${GITEA_REPO:=test/TOIR3}"
: "${GITEA_ACTOR:=xor}"
: "${GITEA_REF_NAME:=v18}"
: "${TEST_TIMEOUT:=60m}"

export GITEA_TOKEN GITEA_URL GITEA_REPO GITEA_ACTOR GITEA_REF_NAME

echo "=== Pipeline Integration Test ==="
echo "  Gitea:  ${GITEA_URL}"
echo "  Repo:   ${GITEA_REPO}"
echo "  Actor:  ${GITEA_ACTOR}"
echo "  Ref:    ${GITEA_REF_NAME}"
echo "  Timeout: ${TEST_TIMEOUT}"
echo ""

cd "$(dirname "$0")/.."

go test -tags=integration ./cmd/apk-ci/ \
  -run TestIntegration_Pipeline_StagesIndividually \
  -v \
  -timeout "${TEST_TIMEOUT}"
