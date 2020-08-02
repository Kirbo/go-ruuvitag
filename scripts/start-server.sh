#!/bin/bash

SCRIPTS_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
BASE_DIR=$(dirname $(echo "${SCRIPTS_DIR}"))
BIN_DIR="${BASE_DIR}/bin"

source "${SCRIPTS_DIR}/_includes/_main.sh"
source "${BASE_DIR}/.env"

cd ${BASE_DIR}
${BIN_DIR}/server
