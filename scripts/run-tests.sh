#!/usr/bin/env bash
set -uo pipefail

# Required env vars
: "${GO_TEST_REPORT_NAME:?GO_TEST_REPORT_NAME must be set}"
: "${GINKGO_TEST_ARGS:?GINKGO_TEST_ARGS must be set}"

# Optional env vars with defaults
GINKGO_PROCS="${GINKGO_PROCS:-4}"
GO_TEST_SKIP_UPLOAD="${GO_TEST_SKIP_UPLOAD:-}"
DATADOG_API_KEY="${DATADOG_API_KEY:-}"
DD_ENV="${DD_ENV:-local}"

SUCCEED_FILE="report-${GO_TEST_REPORT_NAME}-succeed"

# Run the test and touch succeed-file on success
# Do not stop on error (-e is not set)
# shellcheck disable=SC2086
go run github.com/onsi/ginkgo/v2/ginkgo --fail-on-pending --keep-going --vv \
  --cover --coverprofile=cover.profile --randomize-all \
  --race --trace --json-report="report-${GO_TEST_REPORT_NAME}.json" --junit-report="report-${GO_TEST_REPORT_NAME}.xml" \
  --compilers="${GINKGO_PROCS}" --procs="${GINKGO_PROCS}" \
  --poll-progress-after=10s --poll-progress-interval=10s \
  ${GINKGO_TEST_ARGS} \
    && touch "${SUCCEED_FILE}"

# Try upload test reports if allowed and necessary prerequisites exist
if [ "${GO_TEST_SKIP_UPLOAD}" = "true" ]; then
  echo "datadog-ci junit upload SKIPPED"
elif [ -z "${DATADOG_API_KEY}" ]; then
  echo "DATADOG_API_KEY env var is not defined, create a local API key https://app.datadoghq.com/personal-settings/application-keys if you want to upload your local tests results to datadog"
elif ! command -v datadog-ci &>/dev/null; then
  echo "datadog-ci binary is not installed, run 'make install-dev-tools' to upload tests results to datadog"
else
  DD_ENV="${DD_ENV}" datadog-ci junit upload \
    --service chaos-controller \
    --tags="team:chaos-engineering,type:${GO_TEST_REPORT_NAME}" \
    "report-${GO_TEST_REPORT_NAME}.xml" || true
fi

# Fail if succeed file does not exist
if [ -f "${SUCCEED_FILE}" ]; then
  rm -f "${SUCCEED_FILE}"
  exit 0
else
  exit 1
fi
