#!/bin/bash

printUsage() {
  echo -ne "${GREEN}"
 cat << EOF
###############################################################
#
#  Usage:
#      ./scripts/control-ps4.sh [args]
#
#  Parameters:
#
#     Name            Required    Default value     Description
#
#     args            No                            Acceptable value: standby
#
#    Examples:
#       # Turn on the PS4
#           ./scripts/control-ps4.sh
#
#       # Turn off the PS4
#           ./scripts/control-ps4.sh standby
#
###############################################################
EOF
echo -ne "${NORMAL}"
}


SCRIPTS_DIR=$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)
source "${SCRIPTS_DIR}/_includes/_main.sh"

ARGS=$@

/home/pi/.yarn/bin/ps4-waker -c /home/pi/.ps4-wake.credentials.json -d 192.168.1.207 --pass 1337 ${ARGS} >> ~/logs/control-ps4.log
