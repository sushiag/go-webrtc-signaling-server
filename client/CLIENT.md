# WVERTC CLIENT

## Packages

- signaling_client 
- client (client.go, auth_helpers.go)
- peer-manager
- cmd

## client pakage 

This packages contains the interface for WevRTC signaling, room management, peer-to-peer messaging and user authentication for the signaling server.

it contains two parts:

- - user authentication helpers 
- - REST API for the Webrtc signaling client 

this supports the running signaling server:

/ register
/ regenerate
/ newpassword
GET / with API-Key

# struct types

type Client struct {
	sClient *signaling.SignalingClient // WebSocket signaling layer
	pm      *pm.PeerManager            // Handles peer connections
}


- This represents a client connecting to the server, managing rooms, and sending/receivinng peer messages.

type Credentials struct {
	Username string
	Password string
}

- Used for handling username/password combinations.


# Room Management Methods

| Method      | Returns    | Description                                                          |
| Create room | uint64     | Creates a room and then returns `room ID`                            |
| JoinRoom()  | uint 64    | Join an existing room by ID and then returns a list of existing peer |
| Leave room  |            | Leaves Current room                                                  |

# Data Channel Methods

| Method             | Returns                             | Description                               |
| GetDataChOpened()  | chan uint64                         | Channel to notify when peer data is open. |
| GetPeerDataMsg()   | chan pm.PeerDataMsg                 | Channel to notify when peer data is open. |
| SendDataToPeer()   | peerID, uint64, data []byte,  error | Sends data to a peer over WebRTC.         |
| GetClientID()      | Returns the unique client ID        | assigned by the server                    |

## signaling_client

This package implements the websocket-based signaling client used for creating/joining/leaving room in a WebRTC-based p2p communication system.


This contains the following:

- - Authentication by using an API key
- - Handle Messages exchange (create/join/leave room, ping/pong)
- - Maintains Channel for incoming/outgoing signaling messages
- - Establish and manage a websocket connection with the signaling server.

# struct types

type SignalingClient struct {
	ClientID     uint64
	SignalingIn  <-chan smsg.MessageRawJSONPayload
	SignalingOut chan<- smsg.MessageAnyPayload

	createRoom chan smsg.MessageRawJSONPayload
	joinRoom   chan smsg.MessageRawJSONPayload
}

- This represent the core signaling client used to communicate over websockets.

# Websocket internals

- Incoming mesages (readLoop)
- - Listens to JSON messages from the server
- - Handles messages types: ping/pong, RoomCreated, RoomJoined, SignalingIn.

- Outgoing Messages (writeLoop)
- - Sends messages from SignalingOut to the websocket server.
- - Logs each send message type

# Room Management Methods
| Method      | Returns                         | Description                                                                      |
|CreateRoom() | uint64 error                    | sends a CreateRoom message and waits for a Roomcreated or RoomJoinedResponse.    |
|JoinRoom()   | roomID uint64 / []uint64, error | sends a JoinRoom request with room ID, waits for RoomJoined response             |
|LeaveRoom()  |                                 | sends a LeaveRoom message to the server.                                         |

# Signaling Channels 

The Channels allows communication between the client and higher-level logic like the peer manager:

| Channel Name   | Type                         | Description                              |
| SignalingIn    | chan MessageRawJsonPayload   | Receives messages from the server        |
| SignalingOut   | chan MessageAnyPayLoad       | Used to send signaling messages to server|

## peer_manager Package

This package handles the full lifecycle and signaling logic for the Webrtc peer connections in a client. It offers, answer, exhange ICE candidates and peer-states a room of users.

This contains the following:

- Maintains an active maps of peers/clientID.
- sets remotes descriptions
- avoids handling repeats so it deduplicates signaling messages.


# struct types 

type PeerManager struct {
	peers        map[uint64]*peer
	signalingOut chan<- smsg.MessageAnyPayload
	dataChOpened chan uint64
	peerData     chan PeerDataMsg
}

- This represents the PeerManger, a main component that maintains the active WebRTC peers in a map. This sends/receives signaling messages (SDP AND ICE) -- as well as manage evevens for the data channels and icoming data from peers.

type peer struct {
	conn   *webrtc.PeerConnection
	dataCh *webrtc.DataChannel
}

- This represents a single peer connection that includes both the PeerConnection and DataChannel.

type PeerDataMsg {
	From  uint64
	Data {}byte
}

- This represents the data messages from peers, which incluudes the peer sender's ID.

type sendICE struct {
	forPeer uint64
	ice *webrtc.ICECandidate
}

- This represents the internal tracker for the ICE candidates before sending to peers.


# Peer Manager Function

| Function		  	  | Description 																					|
| NewPeerManger() 	  | Initializes the PeerManger, starts the signaling loop and handles incoming signaling messages.	|
| SendDataToPeer  	  | Sends Binary data to the given peer over the data channel.										|
| GetDataChOpenedCh() | Returns a channel that emits peerID's when the data channels are opened.				        |
| GetPeerDataMsg()	  | Returns a channel when rececing data messages from peers.										|

# Notes

- - To start the client, to go `client/cmd/main.go`
- - All HTTP methods use JSON encoding
- - API KEY is required for signaling/ws connection.
- - PeerManger must handle WebRTC peer connection logic internally.
- - This avoids the use of sync related, and locks.
- - Malformed client ID from the server will cause the client to fail upon setup.