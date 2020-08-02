#!/bin/bash

printUsage() {
  echo -ne "${GREEN}"
 cat << EOF
###############################################################
#
#  Usage:
#      ./scripts/create-migration.sh <migration_name>
#
#  Parameters:
#
#     Name            Required    Default value     Description
#
#     migration_name  Yes                           Name of the migration
#
#    Examples:
#       # Create new table
#           ./scripts/create-migration.sh create_users_table
#
#       # Insert test user
#           ./scripts/create-migration.sh insert_test_user
#
#       # Alter users table (add/remove indexes/columns, etc.)
#           ./scripts/create-migration.sh alter_users_table
#
###############################################################
EOF
echo -ne "${NORMAL}"
}


SCRIPTS_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
source "${SCRIPTS_DIR}/_includes/_main.sh"
source "${BASE_DIR}/.env"

cd ${BASE_DIR}

MIGRATION_NAME=$@

if [[ -z "${MIGRATION_NAME}" ]]; then
    printUsage
    exit
fi

migrate create -ext sql -dir ${BASE_DIR}/db/migrations -seq ${MIGRATION_NAME}
