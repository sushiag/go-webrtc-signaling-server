package main

import (
	"encoding/json"
	"log"

	"github.com/pion/webrtc/v4"
)

func handleSignal(msg Message, conn *WebSocketClient) {
	pc, exists := peerManager.Get(msg.Target)
	if !exists {
		var err error
		pc, err = createPeerConnection(msg.Target, conn.Send)
		if err != nil {
			log.Println("Error creating peer connection:", err)
			return
		}
		peerManager.Add(msg.Target, pc)
	}

	switch msg.Type {
	case "offer":
		pc.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  msg.Content,
		})
		answer, _ := pc.CreateAnswer(nil)
		pc.SetLocalDescription(answer)

		conn.Send(Message{
			Type:    "answer",
			Target:  msg.Target,
			Content: answer.SDP,
		})

	case "answer":
		pc.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  msg.Content,
		})

	case "ice-candidate":
		var candidate webrtc.ICECandidateInit
		json.Unmarshal([]byte(msg.Content), &candidate)
		pc.AddICECandidate(candidate)
	}
}
