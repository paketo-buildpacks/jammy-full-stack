#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROG_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly STACK_DIR="$(cd "${PROG_DIR}/.." && pwd)"
readonly BIN_DIR="${STACK_DIR}/.bin"
readonly DEFAULT_BUILD_DIR="${STACK_DIR}/build"

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${PROG_DIR}/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${PROG_DIR}/.util/print.sh"

function main() {
  local build run receiptFilename buildReceipt runReceipt receipts
  build="${DEFAULT_BUILD_DIR}/build.oci"
  run="${DEFAULT_BUILD_DIR}/run.oci"
  receiptFilename="receipt.cyclonedx.json"
  buildReceipt="${DEFAULT_BUILD_DIR}/build-${receiptFilename}"
  runReceipt="${DEFAULT_BUILD_DIR}/run-${receiptFilename}"
  build_image_specified=false

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      --build-image|-b)
        build="${2}"
        build_image_specified=true
        shift 2
        ;;

      --run-image|-r)
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

  # If the build image is specified, then the build directory
  # is differenet from the default
  if [ $build_image_specified = true ]; then
    build_dir=$(dirname "${build}")
  else
    build_dir="${DEFAULT_BUILD_DIR}"
  fi

  # We are generating receipts for all platforms
  receiptFilenames=$(receipts::generate::multi::arch "${build}" "${run}" "${buildReceipt}" "${runReceipt}" "${build_dir}")

  util::print::success "Success! Receipts are:\n${receiptFilenames}"
}

function usage() {
  cat <<-USAGE
receipts.sh [OPTIONS]

Generates receipts listing packages installed on build and run images of the
stack.

OPTIONS
  --help          -h  prints the command usage
  --build-image   -b  path to OCI image of build image. Defaults to
                      ${DEFAULT_BUILD_DIR}/build.oci
  --run-image     -r  path to OCI image of build image
                      ${DEFAULT_BUILD_DIR}/run.oci
  --build-receipt -B  path to output build image package receipt. Defaults to
                      ${DEFAULT_BUILD_DIR}/build-receipt.cyclonedx.json
  --run-receipt   -R  path to output run image package receipt. Defaults to
                      ${DEFAULT_BUILD_DIR}/run-receipt.cyclonedx.json
USAGE
}

function tools::install() {
  util::tools::crane::install \
    --directory "${BIN_DIR}"
  util::tools::jam::install \
    --directory "${BIN_DIR}"
  util::tools::syft::install \
    --directory "${BIN_DIR}"
}

# Generates syft receipts for each architecture for given oci archives
function receipts::generate::multi::arch() {
  local buildArchive runArchive
  local registryPort registryPid localRegistry
  local imageType archiveName imageReceipt receiptFilenames

  receiptFilenames=""
  buildArchive="${1}"
  runArchive="${2}"
  buildOutput="${3}"
  runOutput="${4}"
  build_dir="${5}"

  registryPort=$(get::random::port)
  registryPid=$(local::registry::start $registryPort)
  localRegistry="127.0.0.1:$registryPort"

  # Push the oci archives to the local registry
  jam publish-stack \
    --build-ref "$localRegistry/build" \
    --build-archive $buildArchive \
    --run-ref "$localRegistry/run" \
    --run-archive $runArchive >/dev/null

  # Ensure we can write to the build_dir
  if [ $(stat -c %u $build_dir) = "0" ]; then
    sudo chown -R "$(id -u):$(id -g)" "$build_dir"
  fi

  for archivePath in "${buildArchive}" "${runArchive}" ; do
    archiveName=$(basename "${archivePath}")        # either 'build.oci' or 'run.oci'
    imageType=$(basename -s .oci "${archivePath}")  # either 'build' or 'run'

    util::print::title "Generating package SBOM for ${archiveName}"

    for imageArch in $(crane manifest "$localRegistry/$imageType" | jq -r '.manifests[].platform.architecture'); do

      if [[ "$imageType" = "build" ]]; then
        dir=$(dirname ${buildOutput})
        fileName=$(basename ${buildOutput})
      elif [[ "$imageType" = "run" ]]; then
        dir=$(dirname ${runOutput})
        fileName=$(basename ${runOutput})
      fi

      if [ $imageArch = "amd64" ]; then
        imageReceipt="${dir}/${fileName}"
      else
        imageReceipt="${dir}/${imageArch}-${fileName}"
      fi

      util::print::info "Generating CycloneDX package SBOM using syft for $archiveName on platform linux/$imageArch saved as $imageReceipt"

      # Generate the architecture-specific SBOM from image in the local registry
      syft scan "registry:$localRegistry/$imageType" \
        --output cyclonedx-json="$imageReceipt" \
        --platform "linux/$imageArch"

      receiptFilenames+="$imageReceipt\n"
    done
  done

  kill $registryPid
  echo $receiptFilenames
}

main "${@:-}"
