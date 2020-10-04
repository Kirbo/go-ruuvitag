#!/bin/bash

SCRIPTS_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
BASE_DIR=$(dirname $(echo "${SCRIPTS_DIR}"))
BIN_DIR="${BASE_DIR}/bin"

source "${SCRIPTS_DIR}/_includes/_main.sh"
source "${BASE_DIR}/.env"

USER_LOGS=${1}

cd ${BASE_DIR}
sudo setcap cap_net_raw,cap_net_admin+eip ${BIN_DIR}/client
echo "============= Started: $(date) =============" > ${USER_LOGS}/logs/ruuvi-client
${BIN_DIR}/client 2>&1 | tee -a ${USER_LOGS}/logs/ruuvi-client
