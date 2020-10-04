#!/bin/bash

USER_LOGS=${1}

mkdir -p ${USER_LOGS}/logs

service ruuvitag-client stop
systemctl disable ruuvitag-client.service

cat >/lib/systemd/system/ruuvitag-client.service <<EOL
[Unit]
Description=Go ruuvitag Client
After=multi-user.target

[Service]
Type=simple
ExecStart=${PWD}/scripts/start-client.sh ${USER_LOGS}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOL


chmod 644 /lib/systemd/system/ruuvitag-client.service

systemctl daemon-reload
systemctl enable ruuvitag-client.service
service ruuvitag-client restart
