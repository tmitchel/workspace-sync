package wssync

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pion/webrtc/v2"
	"github.com/sirupsen/logrus"
)

type Remote struct {
	conn *webrtc.PeerConnection
}

func NewRemote(addr string) (*Remote, error) {
	r := &Remote{}
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

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			logrus.Infof("%+v", msg)
			payload := struct {
				Name string
				File []byte
			}{}
			err := json.Unmarshal(msg.Data, &payload)
			if err != nil {
				logrus.Fatal(err)
			}

			err = ioutil.WriteFile(payload.Name, payload.File, 0644)
			if err != nil {
				logrus.Fatal(err)
			}
		})
	})

	// Exchange the offer/answer via HTTP
	offerChan, answerChan := r.mustSignalViaHTTPRemote(addr)

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

func (r *Remote) mustSignalViaHTTPRemote(address string) (chan webrtc.SessionDescription, chan webrtc.SessionDescription) {
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
