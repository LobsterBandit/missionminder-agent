# missionminder-agent

File watcher companion for the [MissionMinder](https://github.com/LobsterBandit/MissionMinder) WoW addon.

The agent monitors `MissionMinder.lua` saved variables file for changes to grab freshly mission data.

## Build

```sh
docker buildx build -t missionminder-agent:latest .
```

## Run

```sh
# e.g. docker run --rm -t -v "/mnt/c/Program Files (x86)/World of Warcraft:/wow" -e TZ=US/Eastern missionminder-agent:latest
docker run --rm -t -v "/your/wow/folder:/wow" -e TZ=<your TZ> missionminder-agent:latest
```
