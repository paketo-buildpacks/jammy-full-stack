#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly STACK_DIR="$(cd "${PROG_DIR}/.." && pwd)"
readonly BIN_DIR="${STACK_DIR}/.bin"
readonly BUILD_DIR="${STACK_DIR}/build"

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${PROG_DIR}/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${PROG_DIR}/.util/print.sh"

function main() {
  local build run buildReceipt runReceipt
  build="${BUILD_DIR}/build.oci"
  run="${BUILD_DIR}/run.oci"
  buildReceipt="${BUILD_DIR}/build-receipt.cyclonedx.json"
  runReceipt="${BUILD_DIR}/run-receipt.cyclonedx.json"

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      --build-archive|-b)
        build="${2}"
        shift 2
        ;;

      --run-archive|-r)
        run="${2}"
        shift 2
        ;;

      --build-receipt|-B)
        buildReceipt="${2}"
        shift 2
        ;;

      --run-receipt|-R)
        runReceipt="${2}"
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

  tools::install

  receipts::generate "${build}" "${buildReceipt}"
  receipts::generate "${run}" "${runReceipt}"

  util::print::success "Success! Receipts are:\n  ${buildReceipt}\n  ${runReceipt}\n"
}

function usage() {
  cat <<-USAGE
receipts.sh [OPTIONS]

Generates receipts listing packages installed on build and run images of the
stack.

OPTIONS
  --help          -h  prints the command usage
  --build-archive -b  path to OCI archive of build image. Defaults to
                      ${BUILD_DIR}/build.oci
  --run-archive   -r  path to OCI archive of build image
                      ${BUILD_DIR}/run.oci
  --build-receipt -B  path to output build image package receipt. Defaults to
                      ${BUILD_DIR}/build-receipt.txt
  --run-receipt   -R  path to output run image package receipt. Defaults to
                      ${BUILD_DIR}/run-receipt.txt
USAGE
}

function tools::install() {
  util::tools::syft::install \
    --directory "${BIN_DIR}"
}

function receipts::generate() {
  local archive output hasDpkg

  archive="${1}"
  output="${2}"

  util::print::title "Generating package SBOM for ${archive}"

  util::print::info "Generating CycloneDX package SBOM using syft"
  syft packages "${archive}" --output cyclonedx-json --file "${output}"
}

main "${@:-}"
