# Workspace Sync

__NOTE:__ I wrote this package for my own needs, so it may not generalize well. Feel free to create issues and PRs, but I may be slow to get to them.

Workplace sync is a tool that allows you to write code in one place and automatically keep a remote copy of the code in sync. [fsnotify](https://github.com/fsnotify/fsnotify) is used to watch the local copy and [Pion](https://github.com/pion/webrtc) Data Channels are used to send a copy of the updated file to the remote when a change is detected.

## Setup

Download the appropriate binary to your local machine and the remote. If one is not provided, compile using
```go
GOOS=<os> GOARCH=<architecture> go build -o sync cmd/workspace-sync/main.go
```

The `config.json` file is used to configure the sync. Here is an example for the local end:
```json
{
    "local": true,
    "directories": "./",
    "ignore": [".git"],
    "port": ":50000",
    "iceURL": "stun:stun.l.google.com:19302",
    "channel": "sync-test"
}
```

This will tell the sync that this is the local end and to watch the current directory (and all subdirectories) ignoring any file with `.git` in the name. `localhost:50000` will be used as the HTTP server for setting up the Data Channel named `sync-test`. Google's STUN server will be used.
