# Example commands

## Create projects dir:
```sh
mkdir -p ${HOME}/projects
cd ${HOME}/projects
git clone git@gitlab.com:kirbo/go-ruuvitag.git
```

## Install / Update Client:
```sh
cd ~/projects/go-ruuvitag/ && git checkout go.* && git pull --rebase && make install && make build-client && make startup-client && make follow-client
```

## Install / Update Server:
```sh
cd ~/projects/go-ruuvitag/ && git checkout go.* && git pull --rebase && make install && make build-server && make startup-server && make follow-server
```
