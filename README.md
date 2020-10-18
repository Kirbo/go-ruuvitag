# Go Ruuvitag Client


## Prequisites

1. Raspberry Pi with Raspbian
2. Ruuvitag Sensors
3. Install Docker, or
    1. Install Go
    2. Install Redis


## Installation

### Installing Redis

```sh
apt install redis
```

### Installing Golang

```sh
make install-go
```

### Installing Docker on Raspbian

https://withblue.ink/2019/07/13/yes-you-can-run-docker-on-raspbian.html#installing-docker

To install Docker CE on Raspbian Stretch and Buster:

```sh
# Install some required packages first
sudo apt update
sudo apt install -y \
     apt-transport-https \
     ca-certificates \
     curl \
     gnupg2 \
     software-properties-common

# Get the Docker signing key for packages
curl -fsSL https://download.docker.com/linux/$(. /etc/os-release; echo "$ID")/gpg | sudo apt-key add -

# Add the Docker official repos
echo "deb [arch=armhf] https://download.docker.com/linux/$(. /etc/os-release; echo "$ID") \
     $(lsb_release -cs) stable" | \
    sudo tee /etc/apt/sources.list.d/docker.list

# Install Docker
# The aufs package, part of the "recommended" packages, won't install on Buster just yet, because of missing pre-compiled kernel modules.
# We can work around that issue by using "--no-install-recommends"
sudo apt update
sudo apt install -y --no-install-recommends \
    docker-ce \
    cgroupfs-mount
```

That’s it! The next step is about starting Docker and enabling it at boot:

```sh
sudo systemctl enable docker
sudo systemctl start docker
```

docker-compose up -d

Other methods to try:
1. First try this: `curl -fsSL https://get.docker.com | bash`, or:
2. https://withblue.ink/2019/07/13/yes-you-can-run-docker-on-raspbian.html
3. https://docs.docker.com/engine/install/debian/


## Configuration

### Configure Redis master

Modify the `/etc/redis/redis.conf´ file.
Comment the line `bind 127.0.0.1 ::1`, around line 50

At the end of it, add the following:
```sh
## Ruuvitag

masterauth passwordForAuthentication
requirepass passwordForAuthentication
```

### Configure Go-Ruuvitag

Copy `example.config.json` as `config.json`:
```sh
cp example.config.json config.json
```
Modify `config.json`


If you want to enable MQTT publishing for RuuviTag sensors, copy `example.mqtt.json` as `mqtt.json`:
```sh
cp example.mqtt.json mqtt.json
```
Modify `mqtt.json`

## Installing Go-Ruuvitag

### Client

Run the following commands:
```sh
make install
make build-client
make startup-client
make follow-client
```

### Server

Run the following commands:
```sh
make install
make build-server
make startup-server
make follow-server
```


## Usage

If you want to see the logs, you can use either:
- `make logs-client`
- `make logs-server`

or:
- `make follow-client`
- `make follow-server`


## Example installation

My setup is as following, I have/use:
- Raspberry Pi 3
- Raspberry Pi 4
- iPad Air 2
- Virtual Server

In more details, on my Raspberry Pi 3:
- I have installed Go RuuviTag Client
- Redis, as replication master
- ["Floorplan"](https://gitlab.com/kirbo/floorplan), that the Go RuuviTag Client is serving in my LAN

On my Raspberry Pi 4:
- I have installed [Home Assistant OS](https://www.home-assistant.io/hassio/)
- MQTT Server

On my Virtual Server:
- I have installed Go RuuviTag Server
- Redis, as replication slave
- ["Floorplan"](https://gitlab.com/kirbo/floorplan), that is served with Nginx, to be accessible everywhere

On my iPad Air 2:
- I open the "floorplan" from my LAN, in Kiosk mode
