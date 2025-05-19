#!/usr/bin/env bash

set -eu
set -o pipefail

readonly SCRIPTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly ROOT_DIR="$(cd "${SCRIPTS_DIR}/.." && pwd)"
readonly STACK_IMAGES_JSON_PATH="${ROOT_DIR}/images.json"
readonly INTEGRATION_JSON="${ROOT_DIR}/integration.json"
declare STACK_IMAGES

# shellcheck source=SCRIPTDIR/.util/tools.sh
source "${SCRIPTS_DIR}/.util/tools.sh"

# shellcheck source=SCRIPTDIR/.util/print.sh
source "${SCRIPTS_DIR}/.util/print.sh"

function main() {
  local clean token test_only_stacks validate_stack_builds
  local registryPort registryPid localRegistry setupLocalRegistry

  help=""
  clean="false"
  token=""
  test_only_stacks=""
  registryPid=""
  setupLocalRegistry=""
  validate_stack_builds="false"

  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        help="true"
        ;;

      --clean|-c)
        shift 1
        clean="true"
        ;;

      --token|-t)
        token="${2}"
        shift 2
        ;;

      --test-only-stacks)
        test_only_stacks="${2}"
        shift 2
        ;;

      --validate-stack-builds)
        validate_stack_builds="true"
        shift 1
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  if [ -f "${STACK_IMAGES_JSON_PATH}" ]; then
    all_stack_images=$(jq -c '.' "${STACK_IMAGES_JSON_PATH}")
  else
    # If there is no images.json file, fallback to the default image configuration
    all_stack_images=$(jq -nc '{
  "images": [
    {
      "config_dir": "stack",
      "output_dir": "build",
      "build_image": "build",
      "run_image": "run",
      "create_build_image": true
    }
  ]
}' | jq -c '.')
  fi

  if [[ -n "${test_only_stacks}" ]]; then
    filter_stacks=$(echo $test_only_stacks | jq -R 'split(" ")')
    STACK_IMAGES=$(echo "${all_stack_images}" | \
      jq --argjson names "$filter_stacks" -c \
      '.images[] | select(.name | IN($names[]))')
    export TEST_ONLY_STACKS="${test_only_stacks}"
  else
    STACK_IMAGES=$(echo "${all_stack_images}" | jq -c '.images[]')
    export TEST_ONLY_STACKS=""
  fi

  ## The help is after the image parsing and filtering in order to output 
  ## proper usage information based on the images that are available
  if [[ "${help}" == "true" ]]; then
    usage
    exit 0
  fi

  tools::install "${token}"

  if [[ "${clean}" == "true" ]]; then
    util::print::title "Cleaning up preexisting stack archives..."
    clean::stacks
  fi

  stack_output_builds_exist=$(stack_builds_exist)

  if [[ "${stack_output_builds_exist}" == "false" && "${validate_stack_builds}" == "true" ]]; then
    util::print::error "Stack builds are not valid."
    exit 1
  fi

  if [[ "${stack_output_builds_exist}" == "false" ]]; then
    util::print::title "Creating stack..."
    while read -r image; do
      config_dir=$(echo "${image}" | jq -r '.config_dir')
      output_dir=$(echo "${image}" | jq -r '.output_dir')
      "${ROOT_DIR}/scripts/create.sh" \
        --stack-dir "${config_dir}" \
        --build-dir "${output_dir}"
    done <<<"$STACK_IMAGES"
  else
    util::print::title "Stack builds already exist..."
  fi

  if [[ -f $INTEGRATION_JSON ]]; then
    setupLocalRegistry=$(jq '.setup_local_registy' $INTEGRATION_JSON)
  fi

  if [[ "${setupLocalRegistry}" == "true" ]]; then
    registryPort=$(get::random::port)
    registryPid=$(local::registry::start $registryPort)
    localRegistry="127.0.0.1:$registryPort"
    export REGISTRY_URL="${localRegistry}"
    util::print::info "Local registry is running with url: ${localRegistry}"
  fi

  tests::run

  if [[ "${setupLocalRegistry}" == "true" ]]; then
    kill $registryPid
  fi
}

function join_by {
  local d=${1-} f=${2-}
  if shift 2; then
    printf %s "$f" "${@/#/$d}"
  fi
}

function usage() {
  oci_images_arr=()

  while read -r image; do
    output_dir=$(echo "${image}" | jq -r '.output_dir')
    build_image=$(echo "${image}" | jq -r '.build_image')
    create_build_image=$(echo "${image}" | jq -r '.create_build_image')
    run_image=$(echo "${image}" | jq -r '.run_image')

    if [ $create_build_image == 'true' ]; then
      oci_images_arr+=("${ROOT_DIR}/${output_dir}/${build_image}.oci")
    fi

    oci_images_arr+=("${ROOT_DIR}/${output_dir}/${run_image}.oci")

  done <<<"$STACK_IMAGES"

  joined_oci_images=$(join_by $'\nand\n' ${oci_images_arr[*]})

  cat <<-USAGE
test.sh [OPTIONS]

Runs acceptance tests against the stack. Uses the OCI images
${joined_oci_images}
if they exist. Otherwise, first runs create.sh to create them.

OPTIONS
  --clean          -c     Clears contents of stack output directory before running tests
  --token <token>  -t     Token used to download assets from GitHub (e.g. jam, pack, etc) (optional)
  --test-only-stacks      Runs the tests of the stacks passed to this argument (e.g. java-8 nodejs-16) (optional)
  --validate-stack-builds Validates that the stack builds are present before running tests (optional)
  --help           -h     Prints the command usage
USAGE
}

function tools::install() {
  local token
  token="${1}"

  util::tools::jam::install \
    --directory "${ROOT_DIR}/.bin" \
    --token "${token}"

  util::tools::pack::install \
    --directory "${ROOT_DIR}/.bin" \
    --token "${token}"

  util::tools::skopeo::check

  util::tools::crane::install \
    --directory "${ROOT_DIR}/.bin" \
    --token "${token}"
}

function tests::run() {
  util::print::title "Run Stack Acceptance Tests"

  export CGO_ENABLED=0
  export JAM_PATH="${ROOT_DIR}/.bin/jam"
  testout=$(mktemp)
  pushd "${ROOT_DIR}" > /dev/null
    if GOMAXPROCS="${GOMAXPROCS:-4}" go test -count=1 -timeout 0 ./... -v -run Acceptance | tee "${testout}"; then
      util::tools::tests::checkfocus "${testout}"
      util::print::success "** GO Test Succeeded **"
    else
      util::print::error "** GO Test Failed **"
    fi
  popd > /dev/null
}

function stack_builds_exist() {

  local stack_output_builds_exist="true"

  while IFS= read -r image; do
    stack_output_dir=$(echo "${image}" | jq -r '.output_dir')
    is_build_image_necessary=$(echo "${image}" | jq -r '.create_build_image // false')

    if ! [[ -f "${ROOT_DIR}/${stack_output_dir}/run.oci" ]]; then
      stack_output_builds_exist="false"
    fi
    if [[ ! -f "${ROOT_DIR}/${stack_output_dir}/build.oci" && "${is_build_image_necessary}" == true ]]; then
      stack_output_builds_exist="false"
    fi
  done <<<"$STACK_IMAGES"

  echo "$stack_output_builds_exist"
}

function clean::stacks(){
  while read -r image; do
    output_dir=$(echo "${image}" | jq -r '.output_dir')
    rm -rf "${ROOT_DIR}/${output_dir}"
  done <<<"$STACK_IMAGES"
}

main "${@:-}"