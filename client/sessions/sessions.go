package client

import (
	"log"

	"github.com/pion/webrtc/v4"
)

func StartSession(apiKey string) (*webrtc.PeerConnection, error) {
	pc, err := NewPeerConnection()
	if err != nil {
		return nil, err
	}

	dc, err := pc.CreateDataChannel("file-transfer", nil)
	if err != nil {
		return nil, err
	}

	dc.OnOpen(func() {
		log.Println("[DATA CHANNEL] Opened")
		// You can now use dc.Send() to send files
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("[DATA CHANNEL] Received %d bytes", len(msg.Data))
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}
	pc.SetLocalDescription(offer)

	answer, err := PostOfferToWHIP(apiKey, offer)
	if err != nil {
		return nil, err
	}

	err = pc.SetRemoteDescription(answer)
	if err != nil {
		return nil, err
	}

	log.Println("[START] Session started successfully")
	return pc, nil
}

func JoinSession(apiKey string) (*webrtc.PeerConnection, error) {
	pc, err := NewPeerConnection()
	if err != nil {
		return nil, err
	}

	offer, err := GetOfferFromWHEP(apiKey)
	if err != nil {
		return nil, err
	}
	pc.SetRemoteDescription(offer)

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}
	pc.SetLocalDescription(answer)

	err = PatchAnswerToWHEP(apiKey, answer)
	if err != nil {
		return nil, err
	}

	log.Println("[JOIN] Joined session successfully")
	return pc, nil
}

func Reconnect(apiKey string) (*webrtc.PeerConnection, error) {
	log.Println("[RECONNECT] Attempting to reconnect...")
	// Can reuse logic from StartSession or JoinSession
	return StartSession(apiKey) // or JoinSession(apiKey)
}

func LeaveSession(sessionID string, direction string) error {
	if direction == "send" {
		return DeleteWHIPSession(sessionID)
	}
	return DeleteWHEPSession(sessionID)
}
