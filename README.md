# Workspace Sync

__NOTE:__ I wrote this package for my own needs, so it may not generalize well. Feel free to create issues and PRs, but I may be slow to get to them.

Workplace sync is a tool that allows you to write code in one place and automatically keep a remote copy of the code in sync. [fsnotify](https://github.com/fsnotify/fsnotify) is used to watch the local copy and [Pion](https://github.com/pion/webrtc) Data Channels are used to send a copy of the updated file to the remote when a change is detected.

## Setup

Download the appropriate binary to your local machine and the remote. If one is not provided, compile using
```go
GOOS=<os> GOARCH=<architecture> go build -o sync cmd/workspace-sync/main.go
```

## Usage - Remote

The remote end must be started first because it runs an HTTP server for the local end to send requests. The remote end is configured using the `config.json` file which must be located at the root of the directory where you want updates to be sent. An example json config is shown below:

```json
{
    "port": ":50000",
    "iceURL": "stun:stun.l.google.com:19302"
}
```

Start the remote end first:
```
./linux-workspace-sync --endpoint remote  // if remote end is linux
```

NOTE: the connection is established via HTTP over localhost. If remote is a server which you SSH into, make sure to forward the port from `config.json`.

## Usage - Local

The local end is configured using a similar `config.json` containing a few extra fields. An example is shown below:
```json
{
    "directories": "./",
    "ignore": [".git"],
    "port": ":50000",
    "iceURL": "stun:stun.l.google.com:19302",
}
```

This this config will recursively watch the current directory and all subdirectories, excluding any paths containing `.git`. Start the local end with:
```
./macos-workspace-sync --endpoint local  // if local end is mac
```

