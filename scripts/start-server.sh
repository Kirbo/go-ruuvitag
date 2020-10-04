#!/bin/bash

SCRIPTS_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
BASE_DIR=$(dirname $(echo "${SCRIPTS_DIR}"))
BIN_DIR="${BASE_DIR}/bin"

source "${SCRIPTS_DIR}/_includes/_main.sh"
source "${BASE_DIR}/.env"

USER_LOGS=${1}

cd ${BASE_DIR}
echo "============= Started: $(date) =============" > ${USER_LOGS}/logs/ruuvi-server
${BIN_DIR}/server 2>&1 | tee -a ${USER_LOGS}/logs/ruuvi-server
