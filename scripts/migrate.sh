#!/bin/bash

printUsage() {
  echo -ne "${GREEN}"
 cat << EOF
###############################################################
#
#  Usage:
#      ./scripts/migrate.sh <operation> [level]
#
#  Parameters:
#
#     Name          Required    Default value     Description
#
#     operation     Yes                           Values: up, down, force
#     level                                       How many migrations do you want to up/down
#
#    Examples:
#       # Run all migrations
#           ./scripts/migrate.sh up
#
#       # Revert all migrations
#           ./scripts/migrate.sh down
#
#       # Revert one migration
#           ./scripts/migrate.sh down 1
#
#       # Force migration to level 1
#           ./scripts/migrate.sh force 1
#
###############################################################
EOF
echo -ne "${NORMAL}"
}


SCRIPTS_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
source "${SCRIPTS_DIR}/_includes/_main.sh"
source "${BASE_DIR}/.env"

cd ${BASE_DIR}

COMMANDS=$@

if [[ -z "${COMMANDS}" ]]; then
    printUsage
    exit
fi

step "Starting migrations"
migrate -path ${BASE_DIR}/db/migrations -database postgres://${POSTGRES_USERNAME}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DATABASE}?sslmode=disable ${COMMANDS}
continue_if_succeeded

echo
step "Running './scripts/generate-sqlc.sh'"
${SCRIPTS_DIR}/generate-sqlc.sh true
continue_if_succeeded

all_done
