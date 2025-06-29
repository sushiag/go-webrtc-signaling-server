package lib

const (
	WSClientJoinedRoom = iota
	WSClientLeftRoom
	WsSDPOffer
	WsSDPResponse
	WsSDPICE
)

type websocketMsg struct {
	msgType  int
	clientID int
	content  string
}

type MockGorillaClient struct{}

func (MockGorillaClient) ReadMessage() (websocketMsg, error) {
	msg := websocketMsg{
		WSClientJoinedRoom, 0, "",
	}
	return msg, nil
}

func (MockGorillaClient) WriteMessage(msg websocketMsg) {}
