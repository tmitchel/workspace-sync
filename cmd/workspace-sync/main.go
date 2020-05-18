package main

import (
	"flag"

	"github.com/sirupsen/logrus"
	wssync "github.com/tmitchel/workspace-sync"
)

func main() {
	local := flag.Bool("local", false, "Run local pion instance")
	addr := flag.String("address", ":50000", "Address to host the HTTP server on.")

	flag.Parse()
	if *local {
		logrus.Fatal(runLocal(*addr))
	}
	logrus.Fatal(runRemote(*addr))
}

func runRemote(addr string) error {
	_, err := wssync.NewRemote(wssync.Config{
		IceURL:      "stun:stun.l.google.com:19302",
		ChannelName: "sync-test",
		Addr:        addr,
	})
	if err != nil {
		return err
	}

	// Block forever
	select {}
}

func runLocal(addr string) error {
	l, err := wssync.NewLocal(wssync.Config{
		IceURL:      "stun:stun.l.google.com:19302",
		ChannelName: "sync-test",
		WatchDir:    []string{"./", "./folder"},
		Addr:        addr,
	})
	if err != nil {
		return err
	}
	defer l.Close()
	go l.Watch()

	// Block forever
	select {}
}
