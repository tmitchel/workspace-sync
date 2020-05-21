package wssync

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
)

// Remote represents the remote copy of the code. This end will
// receive Events and use them to update the remote copy of the
// code.
type Remote struct {
	conn *webrtc.PeerConnection
}

// NewRemote starts handles the RTCPeerConnection and registers
// for receiving events.
func NewRemote(cfg Config) (*Remote, error) {
	r := &Remote{}
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

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		if d.Label() != cfg.ChannelName {
			return
		}

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			var payload Event
			err := json.Unmarshal(msg.Data, &payload)
			if err != nil {
				logrus.Fatal(err)
			}

			dir, _ := filepath.Split(payload.Name)
			if dir != "" {
				os.MkdirAll(dir, 0644)
			}

			err = ioutil.WriteFile(payload.Name, payload.File, 0644)
			if err != nil {
				logrus.Fatal(err)
			}
		})
	})

	// Exchange the offer/answer via HTTP
	offerChan, answerChan := r.mustSignalViaHTTP(cfg.Addr)

	// Wait for the remote SessionDescription
	offer := <-offerChan

	err = peerConnection.SetRemoteDescription(offer)
	if err != nil {
		logrus.Fatal(err)
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		logrus.Fatal(err)
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		logrus.Fatal(err)
	}

	// Send the answer
	answerChan <- answer
	r.conn = peerConnection
	return r, nil
}

// mustSignalViaHTTP handles an incoming offer and returns the answer.
func (r *Remote) mustSignalViaHTTP(address string) (chan webrtc.SessionDescription, chan webrtc.SessionDescription) {
	offerOut := make(chan webrtc.SessionDescription)
	answerIn := make(chan webrtc.SessionDescription)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", http.MethodPost)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Please send a "+http.MethodPost+" request", 400)
			return
		}

		var offer webrtc.SessionDescription
		err := json.NewDecoder(r.Body).Decode(&offer)
		if err != nil {
			logrus.Fatal(err)
		}

		offerOut <- offer
		answer := <-answerIn

		err = json.NewEncoder(w).Encode(answer)
		if err != nil {
			logrus.Fatal(err)
		}
	})

	go func() {
		logrus.Fatal(http.ListenAndServe(address, nil))
	}()
	logrus.Infof("Listening on %v", address)

	return offerOut, answerIn
}
