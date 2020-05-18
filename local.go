package wssync

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

type Local struct {
	watcher *fsnotify.Watcher
	channel *webrtc.DataChannel
}

func NewLocal(addr string) (*Local, error) {
	l := &Local{}
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	dataChannel, err := peerConnection.CreateDataChannel("sync", nil)
	if err != nil {
		return nil, err
	}

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		logrus.Infof("ICE Connection State has changed: %s\n", connectionState.String())
	})

	dataChannel.OnOpen(func() {
		logrus.Infof("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())
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

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		logrus.Infof("Message from DataChannel '%s': '%s'\n", dataChannel.Label(), string(msg.Data))
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
	answer := l.mustSignalViaHTTPLocal(offer, addr)

	// Apply the answer as the remote description
	err = peerConnection.SetRemoteDescription(answer)
	if err != nil {
		return nil, err
	}

	l.channel = dataChannel

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Fatal(err)
	}

	err = watcher.Add("./")
	if err != nil {
		return nil, err
	}

	l.watcher = watcher

	return l, nil
}

func (l *Local) Watch() {
	for {
		select {
		case event, ok := <-l.watcher.Events:
			if !ok {
				return
			}
			logrus.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				logrus.Println("modified file:", event.Name)
				file, err := ioutil.ReadFile(event.Name)
				if err != nil {
					logrus.Fatal(err)
				}

				pl, err := json.Marshal(Event{
					Name: event.Name,
					Op:   event.Op.String(),
					File: file,
				})
				if err != nil {
					logrus.Fatal(err)
				}

				if err := l.channel.Send(pl); err != nil {
					logrus.Fatal(err)
				}
			}
		case err, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
			logrus.Println("error:", err)
		}
	}
}

func (l *Local) mustSignalViaHTTPLocal(offer webrtc.SessionDescription, address string) webrtc.SessionDescription {
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
