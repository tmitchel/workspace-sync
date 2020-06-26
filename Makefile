compile:
	echo Compile for macOS and Linux
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o linux-workspace-sync cmd/workspace-sync/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o macos-workspace-sync cmd/workspace-sync/main.go