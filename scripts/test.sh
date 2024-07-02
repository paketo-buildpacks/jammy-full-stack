#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly STACK_DIR="$(cd "${PROG_DIR}/.." && pwd)"
readonly OUTPUT_DIR="${STACK_DIR}/build"
readonly INTEGRATION_JSON="${STACK_DIR}/integration.json"

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${PROG_DIR}/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${PROG_DIR}/.util/print.sh"

function main() {
  local clean token registryPort registryPid localRegistry setupLocalRegistry
  clean="false"
  token=""
  registryPid=""
  setupLocalRegistry=""

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      --clean|-c)
        shift 1
        clean="true"
        ;;

      --token|-t)
        token="${2}"
        shift 2
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  tools::install "${token}"

  if [[ "${clean}" == "true" ]]; then
    util::print::title "Cleaning up preexisting stack archives..."
    rm -rf "${OUTPUT_DIR}"
  fi

  if ! [[ -f "${OUTPUT_DIR}/build.oci" ]] || ! [[ -f "${OUTPUT_DIR}/run.oci" ]]; then
    util::print::title "Creating stack..."
    "${STACK_DIR}/scripts/create.sh"
  fi

  if [[ -f $INTEGRATION_JSON ]]; then
    setupLocalRegistry=$(jq '.setup_local_registry' $INTEGRATION_JSON)
  fi

  if [[ "${setupLocalRegistry}" == "true" ]]; then
    registryPort=$(get::random::port)
    registryPid=$(local::registry::start $registryPort)
    localRegistry="127.0.0.1:$registryPort"
    export REGISTRY_URL="${localRegistry}"
  fi

  tests::run

  if [[ "${setupLocalRegistry}" == "true" ]]; then
    kill $registryPid
  fi
}

function usage() {
  cat <<-USAGE
test.sh [OPTIONS]

Runs acceptance tests against the stack. Uses the OCI images
${STACK_DIR}/build/build.oci
and
${STACK_DIR}/build/run.oci
if they exist. Otherwise, first runs create.sh to create them.

OPTIONS
  --clean          -c  clears contents of stack output directory before running tests
  --token <token>  -t  Token used to download assets from GitHub (e.g. jam, pack, etc) (optional)
  --help           -h  prints the command usage
USAGE
}

function tools::install() {
  local token
  token="${1}"

  util::tools::jam::install \
    --directory "${STACK_DIR}/.bin" \
    --token "${token}"

  util::tools::pack::install \
    --directory "${STACK_DIR}/.bin" \
    --token "${token}"

  util::tools::skopeo::check

  util::tools::crane::install \
    --directory "${STACK_DIR}/.bin" \
    --token "${token}"
}

function tests::run() {
  util::print::title "Run Stack Acceptance Tests"

  export CGO_ENABLED=0
  testout=$(mktemp)
  pushd "${STACK_DIR}" > /dev/null
    if GOMAXPROCS="${GOMAXPROCS:-4}" go test -count=1 -timeout 0 ./... -v -run Acceptance | tee "${testout}"; then
      util::tools::tests::checkfocus "${testout}"
      util::print::success "** GO Test Succeeded **"
    else
      util::print::error "** GO Test Failed **"
    fi
  popd > /dev/null
}

main "${@:-}"
