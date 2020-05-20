package wssync

// Config provides options for configuring the Local and Remote.
type Config struct {
	// URL for the STUN and TURN server.
	IceURL string

	// Name for the DataChannel that is created. Must match for
	// Local and Remote.
	ChannelName string

	// Directory to watch
	WatchDir string

	// Patterns to ignore
	Ignore []string

	// Address for HTTP exchange of SDP
	Addr string
}

// DefaultConfig returns a default config for watching the current
// directory.
func DefaultConfig() Config {
	return Config{
		IceURL:      "stun:stun.l.google.com:19302",
		ChannelName: "sync",
		WatchDir:    "./",
		Addr:        ":50000",
		Ignore:      []string{".git"},
	}
}
