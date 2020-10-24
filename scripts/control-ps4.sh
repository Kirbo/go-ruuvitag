#!/bin/bash

/home/pi/.yarn/bin/ps4-waker -c /home/pi/.ps4-wake.credentials.json -d 192.168.1.207 --pass 1337 $@ >> ~/logs/control-ps4.log
