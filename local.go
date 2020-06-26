package wssync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

// Local represents the local copy of the code. This is the copy that will
// be edited and sent to the Remote.
type Local struct {
	watcher          *fsnotify.Watcher
	lastOp, lastName string
	channel          *webrtc.DataChannel
}

// Close closes the fsnotify.Watcher.
func (l *Local) Close() error {
	return l.watcher.Close()
}

// NewLocal creates a DataChannel and a new Local watcher then begins
// listening for someone to connect to the channel. On file change events,
// that file will be copied to all clients on the DataChannel.
func NewLocal(cfg Config) (*Local, error) {
	l := &Local{}

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{cfg.IceURL},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	// create the data channel
	dataChannel, err := peerConnection.CreateDataChannel("Workspace-Sync", nil)
	if err != nil {
		return nil, err
	}

	// message on ICE state change
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		logrus.Infof("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// send initial message when channel is opened
	dataChannel.OnOpen(func() {
		logrus.Infof("Data channel '%s'-'%d' open.\n", dataChannel.Label(), dataChannel.ID())
		payload := struct {
			Name string
			File []byte
		}{Name: "connected"}
		pl, err := json.Marshal(payload)
		if err != nil {
			logrus.Fatal(err)
		}
		dataChannel.Send(pl)
	})

	// Create an offer to send to the browser
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		return nil, err
	}

	// Exchange the offer for the answer
	answer := l.mustSignalViaHTTP(offer, cfg.Addr)

	// Apply the answer as the remote description
	err = peerConnection.SetRemoteDescription(answer)
	if err != nil {
		return nil, err
	}
	l.channel = dataChannel

	// create the file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Fatal(err)
	}

	// add all directories to the watcher
	err = filepath.Walk(cfg.WatchDir, func(path string, file os.FileInfo, err error) error {
		for _, f := range cfg.Ignore {
			if strings.Contains(path, f) {
				return nil
			}
		}

		logrus.Infof("Adding directory: %v to the watch", path)
		err = watcher.Add(path)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	l.watcher = watcher

	return l, nil
}

// Watch start the fsnotify.Watcher in an infinite loop watching for
// file events. When it sees one, it send an Event on the DataChannel.
func (l *Local) Watch() {
	for {
		select {
		case event, ok := <-l.watcher.Events:
			if !ok {
				logrus.Error("Event not ok")
				return
			}

			if err := l.handleEvent(event); err != nil {
				logrus.Error(err)
			}
		case err, ok := <-l.watcher.Errors:
			if !ok {
				logrus.Error("Event not ok")
				return
			}
			logrus.Println("error:", err)
		}
	}
}

func (l *Local) handleEvent(event fsnotify.Event) error {
	evt := Event{
		Name: event.Name,
		Op:   event.Op.String(),
	}

	if event.Op&fsnotify.Write == fsnotify.Write {
		logrus.Println("modified file:", event.Name)
		file, err := ioutil.ReadFile(event.Name)
		if err != nil {
			return fmt.Errorf("Error reading file %s : %w", event.Name, err)
		}

		evt.File = file
	} else if event.Op&fsnotify.Rename == fsnotify.Rename {
		if l.lastOp == "CREATE" {
			file, err := ioutil.ReadFile(l.lastName)
			if err != nil {
				return fmt.Errorf("Error reading file %s : %w", event.Name, err)
			}

			// write the old file as the new file
			pl, err := json.Marshal(Event{
				Name: l.lastName,
				Op:   "WRITE",
				File: file,
			})
			if err != nil {
				return fmt.Errorf("Unable to marshal event : %w", err)
			}

			if err := l.channel.Send(pl); err != nil {
				return fmt.Errorf("Unable to send payload : %w", err)
			}
		}
	}

	pl, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("Unable to marshal event : %w", err)
	}

	if err := l.channel.Send(pl); err != nil {
		return fmt.Errorf("Unable to send payload : %w", err)
	}

	l.lastOp = event.Op.String()
	l.lastName = event.Name
	return nil
}

// mustSignalViaHTTP handles sending the SDP to the Remote. Any error is fatal.
func (l *Local) mustSignalViaHTTP(offer webrtc.SessionDescription, address string) webrtc.SessionDescription {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(offer)
	if err != nil {
		logrus.Fatal(err)
	}

	resp, err := http.Post("http://"+address, "application/json; charset=utf-8", b)
	if err != nil {
		logrus.Fatal(err)
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			logrus.Fatal(closeErr)
		}
	}()

	var answer webrtc.SessionDescription
	err = json.NewDecoder(resp.Body).Decode(&answer)
	if err != nil {
		logrus.Fatal(err)
	}

	return answer
}
