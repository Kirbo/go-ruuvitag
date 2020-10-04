#!/bin/bash

USER_LOGS=${1}

mkdir -p ${USER_LOGS}/logs

service ruuvitag-server stop
systemctl disable ruuvitag-server.service

cat >/lib/systemd/system/ruuvitag-server.service <<EOL
[Unit]
Description=Go ruuvitag Server
After=multi-user.target

[Service]
Type=simple
ExecStart=${PWD}/scripts/start-server.sh ${USER_LOGS}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL


chmod 644 /lib/systemd/system/ruuvitag-server.service

systemctl daemon-reload
systemctl enable ruuvitag-server.service
service ruuvitag-server restart
