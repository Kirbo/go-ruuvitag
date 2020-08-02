#!/bin/bash

INCLUDES_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
SCRIPTS_DIR=$(dirname $(echo "${INCLUDES_DIR}"))
BASE_DIR=$(dirname $(echo "${SCRIPTS_DIR}"))

FILENAME=$(basename $(echo ${BASH_SOURCE[0]}))

for FILE in $(ls "${INCLUDES_DIR}" | sort -n | grep -v "${FILENAME}"); do
    source "${INCLUDES_DIR}/${FILE}"
done
